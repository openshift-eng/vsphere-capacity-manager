package resources

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

// CalculateResourceUsage calculates the resource usage of a pool
func CalculateResourceUsage(pool *v1.Pool, lease *v1.Lease) {
	pool.Status.VCpusAvailable -= lease.Spec.VCpus
	pool.Status.MemoryAvailable -= lease.Spec.Memory
	pool.Status.DatastoreAvailable -= lease.Spec.Storage
	pool.Status.NetworkAvailable -= len(pool.Status.PortGroups) - len(pool.Status.ActivePortGroups)
	//pool.Status.NetworkAvailable -= len(lease.Status.PortGroups)
}

func CompareNetworks(a, b v1.Network) bool {
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
			if CompareNetworks(pg, activePg) {
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

// GetFittingPools returns a list of pools that have enough resources to satisfy the resource requirements.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func GetFittingPools(lease *v1.Lease, pools []v1.Pool) []*v1.Pool {
	var fittingPools []*v1.Pool
	for _, pool := range pools {
		if pool.Spec.Exclude {
			if len(lease.Spec.RequiredPool) == 0 || lease.Spec.RequiredPool != pool.ObjectMeta.Name {
				continue
			}
		}
		if int(pool.Status.VCpusAvailable) >= lease.Spec.VCpus &&
			int(pool.Status.MemoryAvailable) >= lease.Spec.Memory &&
			int(pool.Status.DatastoreAvailable) >= lease.Spec.Storage &&
			int(len(pool.Status.PortGroups)-len(pool.Status.ActivePortGroups)) >= lease.Spec.Networks {
			fittingPools = append(fittingPools, &pool)
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
func GetPoolWithStrategy(lease *v1.Lease, pools []v1.Pool, strategy v1.AllocationStrategy) (*v1.Pool, error) {
	fittingPools := GetFittingPools(lease, pools)

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
