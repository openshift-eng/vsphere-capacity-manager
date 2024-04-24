package resources

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	Pools  = map[string]*v1.Pool{}
	Leases = map[string]*v1.Lease{}
	mu     sync.Mutex
)

// CalculateResourceUsage calculates the resource usage of a pool
func CalculateResourceUsage(pool *v1.Pool, leaseMap map[string]*v1.Lease) {
	mu.Lock()
	defer mu.Unlock()

	pool.Status.VCpusAvailable = pool.Spec.VCpus
	pool.Status.MemoryAvailable = pool.Spec.Memory
	pool.Status.DatastoreAvailable = pool.Spec.Storage
	pool.Status.NetworkAvailable = len(pool.Status.PortGroups)

	for _, leaseRef := range pool.Status.Leases {
		if lease, exist := leaseMap[leaseRef.Name]; exist {
			if lease.Status.Pool == nil || lease.Status.Pool.Name != pool.ObjectMeta.Name {
				continue
			}
			pool.Status.VCpusAvailable -= lease.Spec.VCpus
			pool.Status.MemoryAvailable -= lease.Spec.Memory
			pool.Status.DatastoreAvailable -= lease.Spec.Storage
			pool.Status.NetworkAvailable -= len(lease.Status.PortGroups)
		}
	}
}

// AddLease adds a lease to a pool. If the lease already exists in the pool, this function returns
// without error.
func AddLease(lease *v1.Lease) error {
	mu.Lock()
	defer mu.Unlock()

	Leases[lease.ObjectMeta.Name] = lease
	if lease.Status.Pool == nil {
		return fmt.Errorf("lease %s does not have an associated pool", lease.ObjectMeta.Name)
	}
	associatedPoolName := lease.Status.Pool.Name
	if len(lease.Status.Pool.Name) == 0 {
		return fmt.Errorf("lease %s does not have an associated pool", lease.ObjectMeta.Name)
	}

	var pool *v1.Pool
	var exists bool

	if pool, exists = Pools[associatedPoolName]; !exists {
		return fmt.Errorf("pool %s referenced by the lease %s does not exist", associatedPoolName, lease.ObjectMeta.Name)
	}

	for _, l := range pool.Status.Leases {
		if l.Name == lease.Name {
			return nil
		}
	}

	pool.Status.Leases = append(pool.Status.Leases, corev1.TypedLocalObjectReference{
		Name: lease.Name,
	})

	lease.Status.Pool = &corev1.TypedLocalObjectReference{
		Name: pool.ObjectMeta.Name,
	}

	availablePortGroups := GetAvailablePortGroups(pool)
	if len(availablePortGroups) < lease.Spec.Networks {
		return fmt.Errorf("not enough port groups available in pool %s", pool.ObjectMeta.Name)
	}
	lease.Status.PortGroups = availablePortGroups[:lease.Spec.Networks]

	return nil
}

// RemoveLease removes a lease from a pool. If the lease does not exist in the pool, this function
// exits without error.
func RemoveLease(lease *v1.Lease) error {
	mu.Lock()
	defer mu.Unlock()

	associatedPoolName := lease.Status.Pool.Name
	if len(lease.Status.Pool.Name) == 0 {
		return fmt.Errorf("lease %s does not have an associated pool", lease.ObjectMeta.Name)
	}

	var pool *v1.Pool
	var exists bool

	if pool, exists = Pools[associatedPoolName]; !exists {
		return fmt.Errorf("pool %s referenced by the lease %s does not exist", associatedPoolName, lease.ObjectMeta.Name)
	}

	newAvailable := []v1.Network{}
	for _, pg := range lease.Status.PortGroups {
		for _, activePortGroup := range pool.Status.ActivePortGroups {
			if activePortGroup.Network != pg.Network {
				newAvailable = append(newAvailable, pg)
			}
		}
	}
	pool.Status.ActivePortGroups = newAvailable

	for idx, l := range pool.Status.Leases {
		if l.Name == lease.Name {
			pool.Status.Leases = append(pool.Status.Leases[:idx], pool.Status.Leases[idx+1:]...)
			delete(Leases, lease.Name)
			return nil
		}
	}
	return fmt.Errorf("lease %s not found in pool %s", lease.Name, associatedPoolName)
}

func AddPool(pool *v1.Pool) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := Pools[pool.ObjectMeta.Name]; exists {
		return
	}

	pool.Status.VCpusAvailable = pool.Spec.VCpus
	pool.Status.MemoryAvailable = pool.Spec.Memory
	pool.Status.DatastoreAvailable = pool.Spec.Storage
	pool.Status.NetworkAvailable = pool.Spec.Networks

	if pool.Status.Leases == nil {
		pool.Status.Leases = []corev1.TypedLocalObjectReference{}
	}

	ReconcileSubnets([]*v1.Pool{pool})
	Pools[pool.ObjectMeta.Name] = pool
}

func compareNetworks(a, b v1.Network) bool {
	if a.Cidr != b.Cidr ||
		a.CidrIPv6 != b.CidrIPv6 ||
		a.DnsServer != b.DnsServer ||
		a.MachineNetworkCidr != b.MachineNetworkCidr ||
		a.Gateway != b.Gateway ||
		a.Gatewayipv6 != b.Gatewayipv6 ||
		a.Mask != b.Mask ||
		a.Network != b.Network ||
		a.Virtualcenter != b.Virtualcenter ||
		a.Ipv6prefix != b.Ipv6prefix ||
		a.StartIPv6Address != b.StartIPv6Address ||
		a.StopIPv6Address != b.StopIPv6Address ||
		a.LinkLocalIPv6 != b.LinkLocalIPv6 ||
		a.VifIpAddress != b.VifIpAddress ||
		a.VifIPv6Address != b.VifIPv6Address ||
		a.DhcpEndLocation != b.DhcpEndLocation ||
		a.Priority != b.Priority {
		return false
	}

	if !reflect.DeepEqual(a.IpAddresses, b.IpAddresses) {
		return false
	}

	return true
}

// GetAvailablePortGroups returns a list of port groups that are available in a pool
func GetAvailablePortGroups(pool *v1.Pool) []v1.Network {
	availablePortGroups := []v1.Network{}
	for _, pg := range pool.Status.PortGroups {
		inUse := false
		for _, activePg := range pool.Status.ActivePortGroups {
			if compareNetworks(pg, activePg) {
				inUse = true
				break
			}
		}
		if !inUse {
			availablePortGroups = append(availablePortGroups, pg)
		}
	}
	return availablePortGroups
}

// AllocateLease allocates a lease for a pool
// The returned error is really more of a warning. It is not a fatal error if a
// lease cannot be allocated because it has already been allocated.
func AllocateLease(pool *v1.Pool, lease *v1.Lease) error {
	for _, l := range pool.Status.Leases {
		if l.Name == lease.Name {
			return fmt.Errorf("lease %s already exists", lease.Name)
		}
	}
	pool.Status.VCpusAvailable = pool.Status.VCpusAvailable - lease.Spec.VCpus
	pool.Status.MemoryAvailable = pool.Status.MemoryAvailable - lease.Spec.Memory
	pool.Status.DatastoreAvailable = pool.Status.DatastoreAvailable - lease.Spec.Storage
	pool.Status.NetworkAvailable = pool.Status.NetworkAvailable - len(lease.Status.PortGroups)
	pool.Status.Leases = append(pool.Status.Leases, corev1.TypedLocalObjectReference{
		Name: lease.Name,
	})

	/*	for _, portGroup := range lease.Status.PortGroups {
		// Find and remove portGroup from pool.Status.PortGroups
		for i, pg := range pool.Status.PortGroups {
			if compareNetworks(pg, portGroup) {
				pool.Status.PortGroups = append(pool.Status.PortGroups[:i], pool.Status.PortGroups[i+1:]...)
				break
			}
		}

		// Add portGroup to pool.Status.ActivePortGroups
		pool.Status.ActivePortGroups = append(pool.Status.ActivePortGroups, portGroup)
	}*/
	return nil
}

// GetPoolCount returns the number of pools
func GetPoolCount() int {
	mu.Lock()
	defer mu.Unlock()
	return len(Pools)
}

// GetPools returns a list of pools
func GetPools() []*v1.Pool {
	mu.Lock()
	defer mu.Unlock()
	pools := make([]*v1.Pool, len(Pools))
	for _, pool := range Pools {
		pools = append(pools, pool)
	}
	return pools
}

// GetFittingPools returns a list of pools that have enough resources to satisfy the resource requirements.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func GetFittingPools(lease *v1.Lease) []*v1.Pool {
	var fittingPools []*v1.Pool
	for _, pool := range Pools {
		if pool.Spec.Exclude {
			if len(lease.Spec.RequiredPool) == 0 || lease.Spec.RequiredPool != pool.ObjectMeta.Name {
				continue
			}
		}
		if int(pool.Status.VCpusAvailable) >= lease.Spec.VCpus &&
			int(pool.Status.MemoryAvailable) >= lease.Spec.Memory &&
			int(pool.Status.DatastoreAvailable) >= lease.Spec.Storage &&
			int(len(pool.Status.PortGroups)-len(pool.Status.ActivePortGroups)) >= lease.Spec.Networks {
			fittingPools = append(fittingPools, pool)
		}
	}
	sort.Slice(fittingPools, func(i, j int) bool {
		iPool := fittingPools[i]
		jPool := fittingPools[j]
		return iPool.Status.VCpusAvailable+iPool.Status.MemoryAvailable+iPool.Status.DatastoreAvailable+iPool.Status.NetworkAvailable <
			jPool.Status.VCpusAvailable+jPool.Status.MemoryAvailable+jPool.Status.DatastoreAvailable+jPool.Status.NetworkAvailable
	})
	return fittingPools
}

func shuffleFittingPools(pools []*v1.Pool) {
	rand.Shuffle(len(pools), func(i, j int) {
		pools[i], pools[j] = pools[j], pools[i]
	})
}

// GetPoolWithStrategy returns a pool that has enough resources to satisfy the lease requirements.
func GetPoolWithStrategy(lease *v1.Lease, strategy v1.AllocationStrategy) (*v1.Pool, error) {
	fittingPools := GetFittingPools(lease)

	if len(fittingPools) == 0 {
		return nil, fmt.Errorf("no pools with enough resources")
	}
	switch strategy {
	case v1.RESOURCE_ALLOCATION_STRATEGY_RANDOM:
		shuffleFittingPools(fittingPools)
		fallthrough
	case v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED:
		fallthrough
	default:
		return fittingPools[0], nil
	}
}
