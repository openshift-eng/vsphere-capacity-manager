package data

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type AllocationStrategy string

const (
	RESOURCE_ALLOCATION_STRATEGY_RANDOM        = AllocationStrategy("random")
	RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED = AllocationStrategy("under-utilized")
)

// Resource defines the resource requirements for a CI job
type Resource struct {
	Spec              ResourceSpec   `json:"spec"`
	Status            ResourceStatus `json:"status"`
	metav1.ObjectMeta `json:"metadata"`
	metav1.TypeMeta   `json:"type"`
}

type ResourceSpec struct {
	// VCpus is the number of virtual CPUs
	VCpus int `json:"vcpus"`
	// Memory is the amount of memory in GB
	Memory int `json:"memory"`
	// Storage is the amount of storage in GB
	Storage int `json:"storage"`
	// VCenters is the number of vCenters
	VCenters int `json:"vcenters"`
	// Networks is the number of networks requested
	Networks int `json:"networks"`
}

type ResourceStatus struct {
	// Leases is the list of leases assigned to this resource
	Lease Leases `json:"leases"`

	// PortGroups is the list of port groups assigned to this resource
	PortGroups []Network `json:"port-groups"`
}

// Resources is a list of resources
type Resources []Resource
