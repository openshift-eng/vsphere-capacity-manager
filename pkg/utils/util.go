package utils

import (
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DoesLeaseHavePool returns true if a lease already has an associated pool
func DoesLeaseHavePool(lease *v1.Lease) *metav1.OwnerReference {
	var ref *metav1.OwnerReference
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Pool" {
			ref = &ownerRef
		}
	}
	return ref
}
