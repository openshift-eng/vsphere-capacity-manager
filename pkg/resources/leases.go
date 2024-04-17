package resources

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/openshift-splat-team/vsphere-capacity-manager/data"
)

func getLeaseName() string {
	base := "vsphere-lease"
	randomNumber := rand.Int63()
	timestamp := time.Now().UnixNano()
	suffix := fmt.Sprintf("%s-%d-%d", base, timestamp, randomNumber)
	return suffix
}

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
			Status: data.LeaseStatus{
				LeasedAt: time.Now().String(),
				Pool:     pool.Spec.Name,
				Resource: resource,
			},
		}
		pools[idx].Status.Leases = append(pools[idx].Status.Leases, lease)
		leases = append(leases, lease)
	}
	return &leases, nil
}
