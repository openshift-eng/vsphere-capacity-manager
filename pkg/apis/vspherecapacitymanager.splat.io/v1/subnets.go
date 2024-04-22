package v1

// Network
type Network struct {
	Cidr               int      `json:"cidr"`
	CidrIPv6           int      `json:"cidrIPv6"`
	DnsServer          string   `json:"dnsServer"`
	MachineNetworkCidr string   `json:"machineNetworkCidr"`
	Gateway            string   `json:"gateway"`
	Gatewayipv6        string   `json:"gatewayipv6"`
	Mask               string   `json:"mask"`
	Network            string   `json:"network"`
	IpAddresses        []string `json:"ipAddresses"`
	Virtualcenter      string   `json:"virtualcenter"`
	Ipv6prefix         string   `json:"ipv6prefix"`
	StartIPv6Address   string   `json:"startIPv6Address"`
	StopIPv6Address    string   `json:"stopIPv6Address"`
	LinkLocalIPv6      string   `json:"linkLocalIPv6"`
	VifIpAddress       string   `json:"vifIpAddress"`
	VifIPv6Address     string   `json:"vifIPv6Address"`
	DhcpEndLocation    int      `json:"dhcpEndLocation"`
	Priority           int      `json:"priority"`
}

type Subnets map[string]map[string]Network
