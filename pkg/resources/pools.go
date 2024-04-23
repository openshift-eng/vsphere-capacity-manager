package resources

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

var (
	Pools  map[string]*v1.Pool
	Leases v1.Leases
	mu     sync.Mutex
)

// GetPoolByName returns a pool by name
func GetPoolByName(name string) *v1.Pool {
	for _, pool := range Pools {
		if pool.ObjectMeta.Name == name {
			return pool
		}
	}
	return nil
}

func AddPool(pool *v1.Pool) {
	mu.Lock()
	defer mu.Unlock()
	if Pools == nil {
		Pools = make(map[string]*v1.Pool)
	}
	ReconcileSubnets([]*v1.Pool{pool})
	Pools[pool.ObjectMeta.Name] = pool
}

func calculateResourceUsage() {
	log.Printf("calculating pool resource usage")
	for _, pool := range Pools {
		pool.Status.VCpusAvailable = 0
		pool.Status.MemoryAvailable = 0
		pool.Status.DatastoreAvailable = 0
		pool.Status.NetworkAvailable = 0
		for _, lease := range pool.Status.Leases {
			pool.Status.VCpusAvailable += lease.Status.VCpus
			pool.Status.MemoryAvailable += lease.Status.Memory
			pool.Status.DatastoreAvailable += lease.Status.Storage
		}
		pool.Status.VCpusAvailable = pool.Spec.VCpus - pool.Status.VCpusAvailable
		pool.Status.MemoryAvailable = pool.Spec.Memory - pool.Status.MemoryAvailable
		pool.Status.DatastoreAvailable = pool.Spec.Storage - pool.Status.DatastoreAvailable
		pool.Status.NetworkAvailable = len(pool.Status.PortGroups)
		log.Printf("Pool %s Usage: vcpu-available: %d, memory-available: %d, storage-available: %d, network-available: %d",
			pool.ObjectMeta.Name, pool.Status.VCpusAvailable, pool.Status.MemoryAvailable, pool.Status.DatastoreAvailable, pool.Status.NetworkAvailable)
	}
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
	calculateResourceUsage()
	pools := make([]*v1.Pool, len(Pools))
	for _, pool := range Pools {
		pools = append(pools, pool)
	}
	return pools
}

// getFittingPools returns a list of pools that have enough resources to satisfy the resource requirements.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func getFittingPools(resource *v1.ResourceRequestSpec) v1.Pools {
	var fittingPools v1.Pools
	for _, pool := range Pools {
		if pool.Spec.Exclude {
			if len(resource.RequiredPool) == 0 || resource.RequiredPool != pool.ObjectMeta.Name {
				continue
			}
		}
		if int(pool.Status.VCpusAvailable) >= resource.VCpus &&
			int(pool.Status.MemoryAvailable) >= resource.Memory &&
			int(pool.Status.DatastoreAvailable) >= resource.Storage &&
			int(pool.Status.NetworkAvailable) >= resource.Networks {
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

func shuffleFittingPools(pools v1.Pools) {
	rand.Shuffle(len(pools), func(i, j int) {
		pools[i], pools[j] = pools[j], pools[i]
	})
}

func getPoolsWithStrategy(resource *v1.ResourceRequestSpec, strategy v1.AllocationStrategy) (v1.Pools, error) {
	fittingPools := getFittingPools(resource)

	if len(fittingPools) == 0 {
		return nil, fmt.Errorf("no pools with enough resources")
	}
	if len(fittingPools) < resource.VCenters {
		return nil, fmt.Errorf("required number of vCenters exceeds the number of fitting pools")
	}
	switch strategy {
	case v1.RESOURCE_ALLOCATION_STRATEGY_RANDOM:
		shuffleFittingPools(fittingPools)
		return fittingPools[:resource.VCenters], nil
	case v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED:
		fallthrough
	default:
		return fittingPools[:resource.VCenters], nil
	}
}
