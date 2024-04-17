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

func AcquireLease(resource data.Resource) (*data.Leases, error) {
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
		lease := &data.Lease{
			Spec: data.LeaseSpec{
				ResourceSpec: data.ResourceSpec{
					VCpus:   resource.Spec.VCpus,
					Memory:  resource.Spec.Memory,
					Storage: resource.Spec.Storage,
				},
			},
			Status: data.LeaseStatus{
				LeasedAt:  time.Now().String(),
				Pool:      pool.Spec.Name,
				Resources: []data.Resource{resource},
			},
		}
		pools[idx].Status.Leases = append(pools[idx].Status.Leases, lease)
		leases = append(leases, lease)
	}
	return &leases, nil
}
