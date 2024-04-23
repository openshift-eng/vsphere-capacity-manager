package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AllocationStrategy string

const (
	RESOURCE_ALLOCATION_STRATEGY_RANDOM        = AllocationStrategy("random")
	RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED = AllocationStrategy("under-utilized")

	PHASE_FULFILLED Phase = "Fulfilled"
	PHASE_PENDING   Phase = "Pending"
	PHASE_FAILED    Phase = "Failed"
)

type Phase string
type State string

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

// ResourceRequestSpec defines the specification for a resource request
type ResourceRequestSpec struct {
	// VCpus is the number of virtual CPUs
	// defaults to 24 which is a typical vCPU count for a 3 control plane node cluster with
	// 3 compute nodes.
	// +optional
	// +kubebuilder:default=24
	VCpus int `json:"vcpus"`
	// Memory is the amount of memory in GB
	// defaults to 96 which is a typical memory consumption for a 3 control plane node cluster with
	// 3 compute nodes.
	// +optional
	// +kubebuilder:default=96
	Memory int `json:"memory"`
	// Storage is the amount of storage in GB
	// defaults to 720 which is a typical storage consumption for a 3 control plane node cluster with
	// 3 compute nodes.
	// +optional
	// +kubebuilder:default=720
	Storage int `json:"storage"`
	// VCenters is the number of vCenters
	// +optional
	// +kubebuilder:default=1
	VCenters int `json:"vcenters"`
	// Networks is the number of networks requested
	// +optional
	// +kubebuilder:default=1
	Networks int `json:"networks"`
	// RequiredPool when configured, this lease can only be
	// scheduled in the required pool.
	// +optional
	RequiredPool string `json:"required-pool,omitempty"`
}

type ResourceRequestStatus struct {
	// Leases is the list of leases assigned to this resource
	Leases []corev1.TypedLocalObjectReference `json:"leases,omitempty"`

	// Phase is the current phase of the resource request
	Phase Phase `json:"phase"`

	// State is the current state of the resource request
	State State `json:"state"`
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
