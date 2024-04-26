package v1

// Network
type Network struct {
	// Cidr represents the IPv4 network mask.
	// +optional
	Cidr int `json:"cidr"`

	// CidrIPv6 represents the IPv6 network mask.
	// +optional
	CidrIPv6 int `json:"cidrIPv6"`

	// DnsServer represents the DNS server IP address.
	// +optional
	DnsServer string `json:"dnsServer"`

	// MachineNetworkCidr represents the machine network CIDR.
	// +optional
	MachineNetworkCidr string `json:"machineNetworkCidr"`

	// Gateway represents the gateway IP address.
	// +optional
	Gateway string `json:"gateway"`

	// Gatewayipv6 represents the IPv6 gateway IP address.
	// +optional
	Gatewayipv6 string `json:"gatewayipv6"`

	// Mask represents the network mask.
	// +optional
	Mask string `json:"mask"`

	// Network represents the network name.
	// +optional
	Network string `json:"network"`

	// IpAddresses represents a list of IP addresses.
	// +optional
	IpAddresses []string `json:"ipAddresses"`

	// Virtualcenter represents the virtual center server.
	// +optional
	Virtualcenter string `json:"virtualcenter"`

	// Ipv6prefix represents the IPv6 prefix.
	// +optional
	Ipv6prefix string `json:"ipv6prefix"`

	// StartIPv6Address represents the start IPv6 address for DHCP.
	// +optional
	StartIPv6Address string `json:"startIPv6Address"`

	// StopIPv6Address represents the stop IPv6 address for DHCP.
	// +optional
	StopIPv6Address string `json:"stopIPv6Address"`

	// LinkLocalIPv6 represents the link local IPv6 address.
	// +optional
	LinkLocalIPv6 string `json:"linkLocalIPv6"`

	// VifIpAddress represents the VIF IP address.
	// +optional
	VifIpAddress string `json:"vifIpAddress"`

	// VifIPv6Address represents the VIF IPv6 address.
	// +optional
	VifIPv6Address string `json:"vifIPv6Address"`

	// DhcpEndLocation represents the DHCP end location.
	// +optional
	DhcpEndLocation int `json:"dhcpEndLocation"`

	// Priority represents the priority of the network.
	// +optional
	Priority int `json:"priority"`
}

type Subnets map[string]map[string]Network
