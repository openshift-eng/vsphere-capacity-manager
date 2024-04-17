package data

// Pool defines a pool of resources defined available for a given vCenter, cluster, and datacenter
type Pool struct {
	Spec   PoolSpec   `json:"spec"`
	Status PoolStatus `json:"status"`
}
type PoolSpec struct {
	ResourceSpec
	Name       string   `json:"name"`
	VCenter    string   `json:"vcenter"`
	Datacenter string   `json:"datacenter"`
	Cluster    string   `json:"cluster"`
	Datastore  string   `json:"datastore"`
	Networks   []string `json:"networks"`
}

type PoolStatus struct {
	VCpusUsage     float64 `json:"vcpus-usage"`
	MemoryUsage    float64 `json:"memory-usage"`
	DatastoreUsage float64 `json:"datastore-usage"`
	Leases         Leases  `json:"leases"`
	NetworkUsage   float64 `json:"network-usage"`
}

type Pools []*Pool
