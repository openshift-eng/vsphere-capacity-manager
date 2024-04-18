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
	VCenter    string `json:"vcenter"`
	Datacenter string `json:"datacenter"`
	Cluster    string `json:"cluster"`
	Datastore  string `json:"datastore"`
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
