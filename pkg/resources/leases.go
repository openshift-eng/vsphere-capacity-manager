package resources

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getLeaseName() string {
	base := "vsphere-lease"
	randomNumber := rand.Int63()
	timestamp := time.Now().UnixNano()
	suffix := fmt.Sprintf("%s-%d-%d", base, timestamp, randomNumber)
	return suffix
}

func ReleaseLease(leases []v1.Lease) error {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("releasing lease: %v", leases)

	for _, lease := range leases {
		for poolKey, pool := range Pools { // search all pools for the leases
			for idx, l := range pool.Status.Leases {
				if l.Name == lease.Name {
					poolName := lease.Status.Pool.Name
					pool.Status.Leases = append(Pools[poolName].Status.Leases[:idx], Pools[poolName].Status.Leases[idx+1:]...)
					pool.Status.PortGroups = append(Pools[poolName].Status.PortGroups[:idx], Pools[poolName].Status.PortGroups[idx+1:]...)
					//Pools[lease.Status.Pool.Name].Status.Leases = append(Pools[poolIdx].Status.Leases[:idx], Pools[poolIdx].Status.Leases[idx+1:]...)
					//Pools[lease.Status.Pool.Name].Status.PortGroups = append(Pools[poolIdx].Status.PortGroups, lease.Status.PortGroups...)
					newAvailable := []v1.Network{}
					for _, pg := range lease.Status.PortGroups {
						for _, activePortGroup := range pool.Status.ActivePortGroups {
							if activePortGroup.Network != pg.Network {
								newAvailable = append(newAvailable, pg)
							}
						}
					}
					pool.Status.ActivePortGroups = newAvailable
					//Pools[lease.Status.Pool.Name].Status.ActivePortGroups = newAvailable
				}
			}
			Pools[poolKey] = pool
		}
	}
	calculateResourceUsage()
	return nil
}

// AcquireLease acquires a lease(or leases) for a resource
func AcquireLease(request *v1.ResourceRequest) ([]v1.Lease, error) {
	mu.Lock()
	defer mu.Unlock()
	resourceSpec := &request.Spec
	log.Printf("acquiring lease for resource: %v", resourceSpec)
	calculateResourceUsage()

	pools, err := getPoolsWithStrategy(resourceSpec, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
	if err != nil {
		message := fmt.Sprintf("error acquiring lease: %v", err)
		request.Status.Phase = v1.PHASE_PENDING
		request.Status.State = v1.State(message)
		return nil, errors.New(message)
	}

	log.Printf("available pools: %v", len(pools))

	leases := []v1.Lease{}
	for idx, pool := range pools {
		portGroups := pool.Status.PortGroups[:resourceSpec.Networks]
		pools[idx].Status.PortGroups = pool.Status.PortGroups[resourceSpec.Networks:]
		pools[idx].Status.ActivePortGroups = append(pools[idx].Status.ActivePortGroups, portGroups...)
		lease := &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "vsphere-lease-",
				Namespace:    request.ObjectMeta.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: request.APIVersion,
						Kind:       request.Kind,
						Name:       request.ObjectMeta.Name,
						UID:        request.ObjectMeta.UID,
					},
					{
						APIVersion: v1.GroupVersion.Version,
						Kind:       "Pool",
						Name:       pool.ObjectMeta.Name,
						UID:        pool.ObjectMeta.UID,
					},
				},
			},
			Spec: v1.LeaseSpec{},
			Status: v1.LeaseStatus{
				Pool: corev1.TypedLocalObjectReference{
					Name: pool.ObjectMeta.Name,
				},
				PortGroups: portGroups,
				VCpus:      resourceSpec.VCpus,
				Memory:     resourceSpec.Memory,
				Storage:    resourceSpec.Storage,
			},
		}
		pools[idx].Status.Leases = append(pools[idx].Status.Leases, lease)
		request.Status.Leases = append(request.Status.Leases, corev1.TypedLocalObjectReference{
			Name:     lease.Name,
			APIGroup: &v1.GroupVersion.Group,
			Kind:     "Lease",
		})
		leases = append(leases, *lease)
	}
	//	request.Status.Leases = append(request.Status.Leases, leases...)
	request.Status.Phase = v1.PHASE_FULFILLED
	request.Status.State = v1.State("lease acquired successfully")
	return leases, nil
}
