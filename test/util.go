package test

import (
	"fmt"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type shape int64

const (
	SHAPE_SMALL  = shape(1)
	SHAPE_MEDIUM = shape(10)
	SHAPE_LARGE  = shape(100)
)

type lease struct {
	lease v1.Lease
}

// GetLease returns a Lease object for testing
func GetLease() *lease {
	return &lease{
		lease: v1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "sample-lease-",
				Namespace:    "default",
			},
		},
	}
}

func (r *lease) WithName(name string) *lease {
	r.lease.ObjectMeta.Name = name
	return r
}

func (r *lease) WithShape(shape shape) *lease {
	r.lease.Spec.VCpus = int(16 * int64(shape))
	r.lease.Spec.Memory = int(16 * int64(shape))
	r.lease.Spec.Storage = int(120 * int64(shape))
	r.lease.Spec.Networks = int(1 * int64(shape))

	return r
}

func (r *lease) WithPool(pool string) *lease {
	r.lease.Spec.RequiredPool = pool
	return r
}

func (r *lease) Build() *v1.Lease {
	return &r.lease
}

// DoesLeaseHavePool checks if the lease is reflected in the pool
func DoesLeaseHavePool(lease *v1.Lease) (bool, error) {
	if lease.Status.Phase != v1.PHASE_FULFILLED {
		return false, fmt.Errorf("lease %s has not been fulfilled", lease.Name)
	}

	var ref *metav1.OwnerReference
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Pool" {
			ref = &ownerRef
		}
	}

	if ref == nil {
		return false, fmt.Errorf("failed to find pool owner reference for lease %s", lease.Name)
	}

	return true, nil

}
