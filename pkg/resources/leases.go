package resources

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/openshift-splat-team/vsphere-capacity-manager/data"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getLeaseName() string {
	base := "vsphere-lease"
	randomNumber := rand.Int63()
	timestamp := time.Now().UnixNano()
	suffix := fmt.Sprintf("%s-%d-%d", base, timestamp, randomNumber)
	return suffix
}

func ReleaseLease(leases *data.Leases) error {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("releasing lease: %v", leases)

	for _, lease := range *leases {
		for poolIdx, pool := range Pools { // search all pools for the leases
			for idx, l := range pool.Status.Leases {
				if l.Name == lease.Name {
					Pools[poolIdx].Status.Leases = append(Pools[poolIdx].Status.Leases[:idx], Pools[poolIdx].Status.Leases[idx+1:]...)
					Pools[poolIdx].Status.PortGroups = append(Pools[poolIdx].Status.PortGroups, lease.Status.PortGroups...)
					newAvailable := []data.Network{}
					for _, pg := range lease.Status.PortGroups {
						for _, activePortGroup := range Pools[poolIdx].Status.ActivePortGroups {
							if activePortGroup.Network != pg.Network {
								newAvailable = append(newAvailable, pg)
							}
						}
					}
					Pools[poolIdx].Status.ActivePortGroups = newAvailable
				}
			}
		}
	}
	calculateResourceUsage()
	return nil
}

// AcquireLease acquires a lease(or leases) for a resource
func AcquireLease(resource *data.Resource) (*data.Leases, error) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("acquiring lease for resource: %v", resource)
	calculateResourceUsage()

	pools, err := getPoolsWithStrategy(resource, data.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
	if err != nil {
		return nil, fmt.Errorf("error acquiring lease: %s", err)
	}

	log.Printf("available pools: %v", len(pools))

	leases := data.Leases{}
	for idx, pool := range pools {
		portGroups := pool.Status.PortGroups[:resource.Spec.Networks]
		pools[idx].Status.PortGroups = pool.Status.PortGroups[resource.Spec.Networks:]
		copy(resource.Status.PortGroups, portGroups)
		pools[idx].Status.ActivePortGroups = append(pools[idx].Status.ActivePortGroups, portGroups...)
		lease := &data.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: getLeaseName(),
			},
			Spec: data.LeaseSpec{
				ResourceSpec: resource.Spec,
			},
			Status: data.LeaseStatus{
				LeasedAt:   time.Now().String(),
				Pool:       pool.ObjectMeta.Name,
				PortGroups: portGroups,
			},
		}
		pools[idx].Status.Leases = append(pools[idx].Status.Leases, lease)
		leases = append(leases, lease)
	}
	return &leases, nil
}
