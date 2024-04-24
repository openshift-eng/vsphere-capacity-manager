package resources

import (
	"log"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConstructLeases acquires a lease(or leases) for a resource
func ConstructLeases(request *v1.ResourceRequest) []*v1.Lease {
	resourceSpec := &request.Spec
	log.Printf("constructing lease(es) for resource: %v", resourceSpec)

	/*pools, err := getPoolsWithStrategy(resourceSpec, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
	if err != nil {
		message := fmt.Sprintf("error acquiring lease: %v", err)
		request.Status.Phase = v1.PHASE_PENDING
		request.Status.State = v1.State(message)
		return nil, nil, errors.New(message)
	}

	log.Printf("available pools: %v", len(pools))*/

	leases := []*v1.Lease{}
	//for idx, pool := range pools {
	for idx := 0; idx < request.Spec.Networks; idx++ {
		leases = append(leases, &v1.Lease{
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
					/*{
						APIVersion: v1.GroupVersion.Version,
						Kind:       "Pool",
						Name:       pool.ObjectMeta.Name,
						UID:        pool.ObjectMeta.UID,
					},*/
				},
			},
			Spec: v1.LeaseSpec{
				VCpus:    resourceSpec.VCpus,
				Memory:   resourceSpec.Memory,
				Storage:  resourceSpec.Storage,
				Networks: resourceSpec.Networks,
			},
		})
	}
	// leaseRef := corev1.TypedLocalObjectReference{
	// 	Name:     lease.Name,
	// 	APIGroup: &v1.GroupVersion.Group,
	// 	Kind:     "Lease",
	// }

	// AllocateLease(pools[idx], lease)
	// request.Status.Leases = append(request.Status.Leases, leaseRef)
	// leases = append(leases, *lease)

	//request.Status.Phase = v1.PHASE_FULFILLED
	///request.Status.State = v1.State("lease acquired successfully")
	return leases
}
