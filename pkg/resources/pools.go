package resources

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	Pools  v1.Pools
	Leases v1.Leases
	mu     sync.Mutex
)

func init() {
	Pools = append(Pools, &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pool1",
		},
		Spec: v1.PoolSpec{
			Server:     "vcs8e-vc.ocp2.dev.cluster.com",
			Datacenter: "datacenter1",
			Cluster:    "cluster1",
			Datastore:  "datastore1",
			ResourceRequestSpec: v1.ResourceRequestSpec{
				VCpus:   120,
				Memory:  1600,
				Storage: 10000,
			},
		}})
	Pools = append(Pools, &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pool2",
		},
		Spec: v1.PoolSpec{
			Server:     "v8c-2-vcenter.ocp2.dev.cluster.com",
			Datacenter: "datacenter1",
			Cluster:    "cluster1",
			Datastore:  "datastore1",
			ResourceRequestSpec: v1.ResourceRequestSpec{
				VCpus:   120,
				Memory:  1600,
				Storage: 10000,
			},
		}})
	Pools = append(Pools, &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pool3",
		},
		Spec: v1.PoolSpec{
			Server:     "vcenter.ibmc.devcluster.openshift.com",
			Datacenter: "datacenter2",
			Cluster:    "cluster2",
			Datastore:  "datastore2",
			ResourceRequestSpec: v1.ResourceRequestSpec{
				VCpus:   60,
				Memory:  800,
				Storage: 5000,
			},
		}})
	Pools = append(Pools, &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pool4",
		},
		Spec: v1.PoolSpec{
			Server:     "vcenter.devqe.ibmc.devcluster.openshift.com",
			Datacenter: "datacenter3",
			Cluster:    "cluster3",
			Datastore:  "datastore3",
			ResourceRequestSpec: v1.ResourceRequestSpec{
				VCpus:   40,
				Memory:  600,
				Storage: 1000,
			},
		}})
	reconcileSubnets(Pools)
}

func calculateResourceUsage() {
	log.Printf("calculating pool resource usage")
	for _, pool := range Pools {
		pool.Status.VCpusAvailable = 0
		pool.Status.MemoryAvailable = 0
		pool.Status.DatastoreAvailable = 0
		pool.Status.NetworkAvailable = 0
		for _, lease := range pool.Status.Leases {
			pool.Status.VCpusAvailable += lease.Spec.ResourceRequestSpec.VCpus
			pool.Status.MemoryAvailable += lease.Spec.ResourceRequestSpec.Memory
			pool.Status.DatastoreAvailable += lease.Spec.ResourceRequestSpec.Storage
			pool.Status.NetworkAvailable += lease.Spec.ResourceRequestSpec.Networks
		}
		pool.Status.VCpusAvailable = pool.Spec.VCpus - pool.Status.VCpusAvailable
		pool.Status.MemoryAvailable = pool.Spec.Memory - pool.Status.MemoryAvailable
		pool.Status.DatastoreAvailable = pool.Spec.Storage - pool.Status.DatastoreAvailable
		pool.Status.NetworkAvailable = len(pool.Status.PortGroups)
		log.Printf("Pool %s Usage: vcpu-available: %d, memory-available: %d, storage-available: %d, network-available: %d",
			pool.ObjectMeta.Name, pool.Status.VCpusAvailable, pool.Status.MemoryAvailable, pool.Status.DatastoreAvailable, pool.Status.NetworkAvailable)
	}
}

// GetPools returns a list of pools
func GetPools() v1.Pools {
	mu.Lock()
	defer mu.Unlock()
	calculateResourceUsage()
	pools := make(v1.Pools, len(Pools))
	copy(pools, Pools)
	return pools
}

// getFittingPools returns a list of pools that have enough resources to satisfy the resource requirements.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func getFittingPools(resource *v1.ResourceRequestSpec) v1.Pools {
	var fittingPools v1.Pools
	for _, pool := range Pools {
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
