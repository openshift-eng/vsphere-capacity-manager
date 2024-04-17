package data

// Pool defines a pool of resources defined available for a given vCenter, cluster, and datacenter
type Pool struct {
	Spec   PoolSpec   `json:"spec"`
	Status PoolStatus `json:"status"`
}
type PoolSpec struct {
	ResourceSpec
	Name       string `json:"name"`
	VCenter    string `json:"vcenter"`
	Datacenter string `json:"datacenter"`
	Cluster    string `json:"cluster"`
	Datastore  string `json:"datastore"`
}

type PoolStatus struct {
	VCpusAvailable     float64   `json:"vcpus-usage"`
	MemoryAvailable    float64   `json:"memory-usage"`
	DatastoreAvailable float64   `json:"datastore-usage"`
	Leases             Leases    `json:"leases"`
	NetworkAvailable   float64   `json:"network-usage"`
	PortGroups         []Network `json:"port-groups"`
	ActivePortGroups   []Network `json:"active-port-groups"`
}

type Pools []*Pool
