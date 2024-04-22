package v1

import (
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

	Spec LeaseSpec `json:"spec,omitempty"`
	// +optional
	Status LeaseStatus `json:"status,omitempty"`
}

// LeaseSpec defines the specification for a lease
type LeaseSpec struct {
	ResourceRequestSpec `json:",inline"`
}

// LeaseStatus defines the status for a lease
type LeaseStatus struct {
	LeasedAt      string    `json:"leased-at,omitempty"`
	BoskosLeaseID string    `json:"boskos-lease-id,omitempty"`
	Pool          string    `json:"pool,omitempty"`
	PortGroups    []Network `json:"port-groups,omitempty"`
}

type Leases []*Lease

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type LeaseList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Lease `json:"items"`
}
