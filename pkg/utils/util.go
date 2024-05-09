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

// DoesLeaseHaveNetworks returns true if a lease already has an associated network
func DoesLeaseHaveNetworks(lease *v1.Lease) bool {
	requiredNetworks := lease.Spec.Networks
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Network" {
			requiredNetworks--
		}
	}
	return requiredNetworks == 0
}

func GenerateEnvVars(lease *v1.Lease) string {
	/*
		export vsphere_url="${vsphere_url}"
		export vsphere_cluster="${vsphere_cluster}"
		export vsphere_resource_pool="${vsphere_resource_pool}"
		export dns_server="${dns_server}"
		export cloud_where_run="${cloud_where_run}"
		export vsphere_datacenter="${vsphere_datacenter}"
		export vsphere_datastore="${vsphere_datastore}"
		export vsphere_portgroup="${vsphere_portgroup}"
		export vlanid="${vlanid:-unset}"
		export phydc="${phydc:-unset}"
		export primaryrouterhostname="${primaryrouterhostname:-unset}"
	*/

	return ""
}
