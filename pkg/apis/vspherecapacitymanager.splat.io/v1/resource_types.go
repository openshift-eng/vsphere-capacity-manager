package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AllocationStrategy string

const (
	RESOURCE_ALLOCATION_STRATEGY_RANDOM        = AllocationStrategy("random")
	RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED = AllocationStrategy("under-utilized")
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceRequest defines the resource requirements for a CI job
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:scope=Namespaced
// +kubebuilder:subresource:status
type ResourceRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourceRequestSpec `json:"spec"`
	// +optional
	Status ResourceRequestStatus `json:"status"`
}

type ResourceRequestSpec struct {
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
	// RequiredPool when configured, this lease can only be
	// scheduled in the required pool.
	RequiredPool string `json:"required-pool"`
}

type ResourceRequestStatus struct {
	// Leases is the list of leases assigned to this resource
	Lease Leases `json:"leases"`

	// PortGroups is the list of port groups assigned to this resource
	PortGroups []Network `json:"port-groups"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ResourceRequestList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ResourceRequest `json:"items"`
}

// Resources is a list of resources
type Resources []ResourceRequest
