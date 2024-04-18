package data

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Pool defines a pool of resources defined available for a given vCenter, cluster, and datacenter
type Pool struct {
	Spec              PoolSpec   `json:"spec"`
	Status            PoolStatus `json:"status"`
	metav1.ObjectMeta `json:"metadata"`
	metav1.TypeMeta   `json:"type"`
}
type PoolSpec struct {
	ResourceSpec
	// Server the server that provisions resources for the pool
	Server    string `json:"server"`
	// Datacenter associated with this pool
	Datacenter string `json:"datacenter"`
	// Cluster cluster associated with this pool
	Cluster    string `json:"cluster"`
	// Datastore datastore associated with this pool
	Datastore  string `json:"datastore"`
	// Exclude when true, this pool is excluded from the default pools.
	// This is useful if a job must be scheduled to a specific pool and that
	// pool only has limited capacity.
	Exclude    bool `json:"exclude`
}

type PoolStatus struct {
	VCpusAvailable     int       `json:"vcpus-usage"`
	MemoryAvailable    int       `json:"memory-usage"`
	DatastoreAvailable int       `json:"datastore-usage"`
	Leases             Leases    `json:"leases"`
	NetworkAvailable   int       `json:"network-usage"`
	PortGroups         []Network `json:"port-groups"`
	ActivePortGroups   []Network `json:"active-port-groups"`
}

type Pools []*Pool
