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

	leases := []*v1.Lease{}
	for idx := 0; idx < request.Spec.VCenters; idx++ {
		leases = append(leases, &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "vsphere-lease-",
				Finalizers:   []string{v1.LeaseFinalizer},
				Namespace:    request.ObjectMeta.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: request.APIVersion,
						Kind:       request.Kind,
						Name:       request.ObjectMeta.Name,
						UID:        request.ObjectMeta.UID,
					},
				},
			},
			Spec: v1.LeaseSpec{
				VCpus:        resourceSpec.VCpus,
				Memory:       resourceSpec.Memory,
				Storage:      resourceSpec.Storage,
				Networks:     resourceSpec.Networks,
				RequiredPool: resourceSpec.RequiredPool,
			},
		})
	}

	return leases
}
