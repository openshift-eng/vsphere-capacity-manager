package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	POOLS_LAST_LEASE_UPDATE_ANNOTATION = "vspherecapacitymanager.splat.io/last-pool-update"
	PoolFinalizer                      = "vsphere-capacity-manager.splat-team.io/pool-finalizer"
	PoolKind                           = "Pool"
)

// TaintEffect defines the effect of a taint on pools that do not tolerate the taint.
type TaintEffect string

const (
	// TaintEffectNoSchedule means leases are not scheduled onto pools with this taint
	// unless they tolerate the taint.
	TaintEffectNoSchedule TaintEffect = "NoSchedule"
	// TaintEffectPreferNoSchedule means the scheduler tries to avoid scheduling leases
	// onto pools with this taint, but it's not required.
	TaintEffectPreferNoSchedule TaintEffect = "PreferNoSchedule"
)

// Taint represents a taint that can be applied to a pool.
type Taint struct {
	// Key is the taint key to be applied to a pool.
	Key string `json:"key"`
	// Value is the taint value corresponding to the taint key.
	// +optional
	Value string `json:"value,omitempty"`
	// Effect indicates the effect of the taint on leases that do not tolerate the taint.
	// Valid effects are NoSchedule and PreferNoSchedule.
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule
	Effect TaintEffect `json:"effect"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pool defines a pool of resources defined available for a given vCenter, cluster, and datacenter
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="vCPUs",type=string,JSONPath=`.status.vcpus-available`
// +kubebuilder:printcolumn:name="Memory(GB)",type=string,JSONPath=`.status.memory-available`
// +kubebuilder:printcolumn:name="Networks",type=string,JSONPath=`.status.network-available`
// +kubebuilder:printcolumn:name="Disabled",type=string,JSONPath=`.spec.noSchedule`
// +kubebuilder:printcolumn:name="Excluded",type=string,JSONPath=`.spec.exclude`
type Pool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PoolSpec `json:"spec"`
	// +optional
	Status PoolStatus `json:"status"`
}

type IBMPoolSpec struct {
	// Pod the pod in the datacenter where the vCenter resides
	Pod string `json:"pod"`
	// Pod the pod in the datacenter where the vCenter resides
	Datacenter string `json:"datacenter"`
}

// PoolSpec defines the specification for a pool
type PoolSpec struct {
	FailureDomainSpec `json:",inline"`
	// IBMPoolSpec topology information associated with this pool
	IBMPoolSpec IBMPoolSpec `json:"ibmPoolSpec,omitempty"`
	// VCpus is the number of virtual CPUs
	VCpus int `json:"vcpus"`
	// +kubebuilder:default="1.0"
	OverCommitRatio string `json:"overCommitRatio"`
	// Memory is the amount of memory in GB
	Memory int `json:"memory"`
	// Storage is the amount of storage in GB
	Storage int `json:"storage"`
	// Exclude when true, this pool is excluded from the default pools.
	// This is useful if a job must be scheduled to a specific pool and that
	// pool only has limited capacity.
	Exclude bool `json:"exclude"`
	// NoSchedule when true, new leases for this pool will not be allocated.
	// any in progress leases will remain active until they are destroyed.
	// +optional
	NoSchedule bool `json:"noSchedule"`
	// Taints are taints applied to this pool. Leases will not be scheduled on this pool
	// unless they have matching tolerations. This works like Kubernetes node taints.
	// +optional
	Taints []Taint `json:"taints,omitempty"`
}

// PoolStatus defines the status for a pool
type PoolStatus struct {
	// vcpus-available is the number of vCPUs available in the pool
	// +optional
	VCpusAvailable int `json:"vcpus-available"`
	// memory-available is the amount of memory in GB available in the pool
	// +optional
	MemoryAvailable int `json:"memory-available"`
	// datastore-available is the amount of storage in GB available in the pool
	// +optional
	DatastoreAvailable int `json:"datastore-available"`
	// network-available is the number of networks available in the pool
	// +optional
	NetworkAvailable int `json:"network-available"`
	// lease-count is the number of leases assigned to the pool
	LeaseCount int `json:"lease-count"`

	// Initialized when true, the status fields have been initialized
	// +optional
	Initialized bool `json:"initialized"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PoolList is a list of pools
type PoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pool `json:"items"`
}

type Pools []*Pool
