package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkType string

const (
	LeaseKind               = "Lease"
	APIGroupName            = "vsphere-capacity-manager.splat-team.io"
	LeaseFinalizer          = "vsphere-capacity-manager.splat-team.io/lease-finalizer"
	LeaseNamespace          = "vsphere-capacity-manager.splat-team.io/lease-namespace"
	NetworkTypeDisconnected = NetworkType("disconnected")
	NetworkTypeSingleTenant = NetworkType("single-tenant")
	NetworkTypeMultiTenant  = NetworkType("multi-tenant")
)

// TolerationOperator is the operator for a toleration.
type TolerationOperator string

const (
	// TolerationOpExists means the toleration matches a taint if the key exists.
	TolerationOpExists TolerationOperator = "Exists"
	// TolerationOpEqual means the toleration matches a taint if key and value are equal.
	TolerationOpEqual TolerationOperator = "Equal"
)

// Toleration represents a toleration that allows a lease to be scheduled on a pool with matching taints.
type Toleration struct {
	// Key is the taint key that the toleration applies to. Empty means match all taint keys.
	// If the operator is Exists, the value should be empty, otherwise just a regular key.
	// +optional
	Key string `json:"key,omitempty"`
	// Operator represents the relationship between the key and value.
	// Valid operators are Exists and Equal. Defaults to Equal.
	// Exists is equivalent to wildcard for value, so that a lease can tolerate all taints of a particular category.
	// +kubebuilder:validation:Enum=Exists;Equal
	// +optional
	Operator TolerationOperator `json:"operator,omitempty"`
	// Value is the taint value the toleration matches to.
	// If the operator is Exists, the value should be empty, otherwise just a regular value.
	// +optional
	Value string `json:"value,omitempty"`
	// Effect indicates which taint effect to match. Empty means match all taint effects.
	// When specified, allowed values are NoSchedule and PreferNoSchedule.
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule;""
	// +optional
	Effect string `json:"effect,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Lease represents the definition of resources allocated for a resource pool
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="vCPUs",type=string,JSONPath=`.spec.vcpus`
// +kubebuilder:printcolumn:name="Memory(GB)",type=string,JSONPath=`.spec.memory`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
type Lease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LeaseSpec `json:"spec"`
	// +optional
	Status LeaseStatus `json:"status"`
}

// LeaseSpec defines the specification for a lease
//
// The XValidation rules below address the min-vcenters property as
// min__dash__vcenters: CEL must escape the dash in kebab-case property names.
// +kubebuilder:validation:XValidation:rule="!has(self.min__dash__vcenters) || self.min__dash__vcenters <= self.pools",message="min-vcenters must not exceed pools: each pool resides on exactly one vCenter"
// +kubebuilder:validation:XValidation:rule="!has(self.min__dash__vcenters) || !has(self.vcenters) || self.vcenters == 0 || self.min__dash__vcenters <= self.vcenters",message="min-vcenters must not exceed the vcenters cap"
type LeaseSpec struct {
	// VCpus is the number of virtual CPUs allocated for this lease
	VCpus int `json:"vcpus,omitempty"`
	// Memory is the amount of memory in GB allocated for this lease
	Memory int `json:"memory,omitempty"`
	// Pools is the number of pools to return for this lease
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// +optional
	Pools int `json:"pools,omitempty"`
	// VCenters is the maximum number of distinct vCenters (identified by Server FQDN)
	// to use when fulfilling this lease. When 0 or unset, no limit is applied.
	// This acts as a cap on vcenter diversity across all assigned pools.
	// For example, a lease with pools: 4 and vcenters: 3 will assign 4 pools but
	// draw them from at most 3 distinct vCenters.
	// See MinVCenters for requiring a minimum number of distinct vCenters instead.
	// +kubebuilder:validation:Minimum=0
	// +optional
	VCenters int `json:"vcenters,omitempty"`
	// MinVCenters is the minimum number of distinct vCenters (identified by Server
	// FQDN) required across the pools assigned to this lease. When 0 or unset, no
	// minimum is applied.
	// While the lease's assigned pools span fewer vCenters than this minimum, pool
	// selection is restricted to vCenters not already in use, so the lease cannot
	// be fulfilled from fewer vCenters than required.
	// For example, a lease with pools: 3 and min-vcenters: 2 will be assigned 3
	// pools drawn from at least 2 distinct vCenters.
	// The value must not exceed spec.pools, and when spec.vcenters is set it must
	// not exceed that cap either; the API server rejects leases that violate
	// either rule at admission time (see the XValidation rules on LeaseSpec), and
	// the controller fails any that predate those rules as unschedulable.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MinVCenters int `json:"min-vcenters,omitempty"`
	// Storage is the amount of storage in GB allocated for this lease
	// +optional
	Storage int `json:"storage,omitempty"`
	// Networks is the number of networks requested
	Networks int `json:"networks"`
	// RequiredPool when configured, this lease can only be fulfilled by a specific
	// pool
	// +optional
	RequiredPool string `json:"required-pool,omitempty"`

	// PoolSelector is a label selector for pools. If specified, the lease can only
	// be fulfilled by pools matching all of the specified label key-value pairs.
	// This works like Kubernetes nodeSelector for selecting pools based on labels.
	// +optional
	PoolSelector map[string]string `json:"poolSelector,omitempty"`

	// Tolerations are tolerations that allow this lease to be scheduled on pools with matching taints.
	// This works like Kubernetes pod tolerations for scheduling on nodes with taints.
	// +optional
	Tolerations []Toleration `json:"tolerations,omitempty"`

	// NetworkType defines the type of network required by the lease.
	// by default, all networks are treated as single-tenant. single-tenant networks
	// are only used by one CI jobs.  multi-tenant networks reside on a
	// VLAN which may be used by multiple jobs.  disconnected networks aren't yet
	// supported.
	// +kubebuilder:validation:Enum="";disconnected;single-tenant;multi-tenant;nested-multi-tenant;public-ipv6
	// +kubebuilder:default=single-tenant
	// +optional
	NetworkType NetworkType `json:"network-type"`

	// BoskosLeaseID is the ID of the lease in Boskos associated with this lease
	// +optional
	BoskosLeaseID string `json:"boskos-lease-id,omitempty"`
}

// LeaseStatus defines the status for a lease
type LeaseStatus struct {
	// Deprecated: The inline FailureDomainSpec fields (name, server, region, zone, topology, shortName)
	// are deprecated for multi-pool leases. Use PoolInfo instead, which provides this information
	// for each assigned pool. For backward compatibility, these fields are populated from the first pool.
	FailureDomainSpec `json:",inline"`

	// PoolInfo contains FailureDomainSpec for each pool assigned to this lease.
	// For multi-pool leases, this array will have multiple entries.
	// Each entry contains name, server, region, zone, topology, and shortName for a pool.
	// +optional
	PoolInfo []FailureDomainSpec `json:"poolInfo,omitempty"`

	// EnvVars a freeform string which contains bash which is to be sourced
	// by the holder of the lease.
	// Deprecated: Use EnvVarsMap instead for multi-pool leases
	// +optional
	EnvVars string `json:"envVars,omitempty"`

	// EnvVarsMap contains environment variables for each pool.
	// The key is the pool name and the value is the bash script to be sourced.
	// This field supports multi-pool leases where each pool has different configurations.
	// +optional
	EnvVarsMap map[string]string `json:"envVarsMap,omitempty"`

	// Phase is the current phase of the lease
	// +optional
	Phase Phase `json:"phase,omitempty"`

	// conditions defines the current state of the Machine
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty"`

	// JobLink defines a link to the job that owns this lease.  Its primarily used when debugging issues w/ lease management.
	JobLink string `json:"job-link,omitempty"`
}

type Leases []*Lease

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type LeaseList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Lease `json:"items"`
}
