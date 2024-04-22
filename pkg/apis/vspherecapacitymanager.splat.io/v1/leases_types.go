package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LeaseKind    = "Lease"
	APIGroupName = "vsphere-capacity-manager.splat-team.io"
)

// +genclient
// +genclient:noStatus
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Lease represents the definition of resources allocated for a resource pool
// +kubebuilder:object:root=true
type Lease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +required
	Spec LeaseSpec `json:"spec"`

	// status represents the current information/status for the IP pool.
	// Populated by the system.
	// Read-only.
	// +optional
	Status LeaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type LeaseList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Lease `json:"items"`
}

type Leases []*Lease

type LeaseSpec struct {
	ResourceRequestSpec `json:",inline"`
}

type LeaseStatus struct {
	LeasedAt      string    `json:"leased-at"`
	BoskosLeaseID string    `json:"boskos-lease-id"`
	Pool          string    `json:"pool"`
	PortGroups    []Network `json:"port-groups"`
}
