package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Subnet defines a pool of resources defined available for a given vCenter, cluster, and datacenter
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:scope=Namespaced
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SubnetSpec `json:"spec"`
	// +optional
	Status SubnetStatus `json:"status"`
}

// SubnetSpec defines the specification for a pool
type SubnetSpec struct {
	// PortGroupName is the non-pathed network (port group) name
	PortGroupName string `json:"portGroupName"`

	// PortGroupFullPath is the govmomi-based full path of the network (port group) object
	PortGroupFullPath string `json:"portGroupFullPath"`

	VlanId string `json:"vlanId"`

	// The PodName is the pod that this VLAN is associated with.
	PodName *string `json:"podName,omitempty"`
	// DatacenterName is foo

	// The DatacenterName is the datacenter that the firewall resides in.
	DatacenterName *string `json:"datacenterName"`

	// The Classless Inter-Domain Routing prefix of this subnet, which specifies the range of spanned IP addresses.
	//
	// [Classless_Inter-Domain_Routing at Wikipedia](http://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing)
	Cidr *int `json:"cidr,omitempty"`

	// The IP address of this subnet reserved for use on the router as a gateway address and which is unavailable for other use.
	Gateway *string `json:"gateway,omitempty"`

	// A count of the IP address records belonging to this subnet.
	IpAddressCount *uint `json:"ipAddressCount,omitempty"`

	// The bitmask in dotted-quad format for this subnet, which specifies the range of spanned IP addresses.
	Netmask *string `json:"netmask,omitempty"`

	SubnetType *string `json:"subnetType,omitempty"`

	// MachineNetworkCidr represents the machine network CIDR.
	// +optional
	MachineNetworkCidr string `json:"machineNetworkCidr"`

	// The IP address records belonging to this subnet.
	// +optional
	IpAddresses []string `json:"ipAddresses"`

	// CidrIPv6 represents the IPv6 network mask.
	// +optional
	CidrIPv6 int `json:"cidrIPv6"`
	// GatewayIPv6 represents the IPv6 gateway IP address.
	// +optional
	GatewayIPv6 string `json:"gatewayipv6"`

	// Ipv6prefix represents the IPv6 prefix.
	// +optional
	IpV6prefix string `json:"ipv6prefix"`

	// StartIPv6Address represents the start IPv6 address for DHCP.
	// +optional
	StartIPv6Address string `json:"startIPv6Address"`
}

// SubnetStatus defines the status for a pool
type SubnetStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubnetList is a list of pools
type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Subnet `json:"items"`
}

type Subnets []*Subnet
