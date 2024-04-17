package v1

import (
	"github.com/openshift-splat-team/vsphere-capacity-manager/data"
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
type Lease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +required
	Spec data.LeaseSpec `json:"spec"`

	// status represents the current information/status for the IP pool.
	// Populated by the system.
	// Read-only.
	// +optional
	Status data.LeaseSpec `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type LeaseList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Lease `json:"items"`
}
