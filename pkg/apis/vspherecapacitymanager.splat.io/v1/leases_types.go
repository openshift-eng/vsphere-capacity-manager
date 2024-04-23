package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LeaseKind    = "Lease"
	APIGroupName = "vsphere-capacity-manager.splat-team.io"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Lease represents the definition of resources allocated for a resource pool
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:scope=Namespaced
// +kubebuilder:subresource:status
type Lease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LeaseSpec `json:"spec"`
	// +optional
	Status LeaseStatus `json:"status"`
}

// LeaseSpec defines the specification for a lease
type LeaseSpec struct{}

// LeaseStatus defines the status for a lease
type LeaseStatus struct {
	// VCpus is the number of virtual CPUs allocated for this lease
	// +optional
	VCpus int `json:"vcpus,omitempty"`
	// Memory is the amount of memory in GB allocated for this lease
	// +optional
	Memory int `json:"memory,omitempty"`
	// Storage is the amount of storage in GB allocated for this lease
	// +optional
	Storage int `json:"storage,omitempty"`

	// Pool is the pool from which the lease was acquired
	// +optional
	Pool corev1.TypedLocalObjectReference `json:"pool,omitempty"`

	// BoskosLeaseID is the ID of the lease in Boskos associated with this lease
	// +optional
	BoskosLeaseID string `json:"boskos-lease-id,omitempty"`

	// PortGroups is the list of port groups associated with this lease
	// +optional
	PortGroups []Network `json:"port-groups,omitempty"`
}

type Leases []*Lease

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type LeaseList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Lease `json:"items"`
}
