package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pool defines a pool of resources defined available for a given vCenter, cluster, and datacenter
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:scope=Namespaced
// +kubebuilder:subresource:status
type Pool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PoolSpec `json:"spec"`
	// +optional
	Status PoolStatus `json:"status"`
}

// PoolSpec defines the specification for a pool
type PoolSpec struct {
	// VCpus is the number of virtual CPUs
	VCpus int `json:"vcpus"`
	// Memory is the amount of memory in GB
	Memory int `json:"memory"`
	// Storage is the amount of storage in GB
	Storage int `json:"storage"`
	// Networks is the number of networks requested
	Networks int `json:"networks"`
	// Server the server that provisions resources for the pool
	Server string `json:"server"`
	// Datacenter associated with this pool
	Datacenter string `json:"datacenter"`
	// Cluster cluster associated with this pool
	Cluster string `json:"cluster"`
	// Datastore datastore associated with this pool
	Datastore string `json:"datastore"`
	// Exclude when true, this pool is excluded from the default pools.
	// This is useful if a job must be scheduled to a specific pool and that
	// pool only has limited capacity.
	Exclude bool `json:"exclude"`
}

// PoolStatus defines the status for a pool
type PoolStatus struct {
	// VCPUsAvailable is the number of vCPUs available in the pool
	VCpusAvailable int `json:"vcpus-available"`
	// MemoryAvailable is the amount of memory in GB available in the pool
	MemoryAvailable int `json:"memory-available"`
	// StorageAvailable is the amount of storage in GB available in the pool
	DatastoreAvailable int `json:"datastore-available"`
	// Leases is the list of leases assigned to this pool
	Leases Leases `json:"leases"`
	// Networks is the number of networks available in the pool
	NetworkAvailable int `json:"network-available"`
	// PortGroups is the list of port groups available in the pool
	PortGroups []Network `json:"port-groups"`
	// ActivePortGroups is the list of port groups that are currently in use
	ActivePortGroups []Network `json:"active-port-groups"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PoolList is a list of pools
type PoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pool `json:"items"`
}

type Pools []*Pool
