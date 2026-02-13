package utils

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

var (
	parsedTemplate *template.Template
)

func init() {
	var err error
	sourceTemplate := `export vsphere_url="{{.Server}}"
		export GOVC_URL="{{.Server}}"
		export GOVC_DATACENTER="{{.VDatacenter}}"
		export GOVC_DATASTORE="{{.Datastore}}"
		export GOVC_NETWORK="{{.PortGroup}}"
		export vsphere_cluster="{{.ComputeCluster}}"
		export vsphere_resource_pool="{{.ResourcePool}}"
		export vsphere_datacenter="{{.VDatacenter}}"
		export vsphere_datastore="{{.Datastore}}"
		export vsphere_portgroup="{{.PortGroup}}"
		export gateway="{{.Gateway}}"
		export dns_server="{{.Nameserver}}"
		export vlanid="{{.VlanId}}"
		export phydc="{{.IDatacenter}}"
		export primaryrouterhostname="{{.PrimaryRouterHostname}}"`

	parsedTemplate, err = template.New("source").Parse(sourceTemplate)
	if err != nil {
		panic(err)
	}
}

// DoesLeaseHavePool returns true if a lease already has an associated pool
func DoesLeaseHavePool(lease *v1.Lease) *metav1.OwnerReference {
	var ref *metav1.OwnerReference
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Pool" {
			ref = &ownerRef
		}
	}
	return ref
}

// GetLeasePoolRefs returns all pool owner references for a lease
func GetLeasePoolRefs(lease *v1.Lease) []metav1.OwnerReference {
	var poolRefs []metav1.OwnerReference
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Pool" {
			poolRefs = append(poolRefs, ownerRef)
		}
	}
	return poolRefs
}

// DoesLeaseHaveAllPools checks if a lease has all required pools assigned
func DoesLeaseHaveAllPools(lease *v1.Lease) bool {
	requiredPools := lease.Spec.Pools
	if requiredPools == 0 {
		requiredPools = 1 // default to 1 pool
	}
	poolCount := 0
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Pool" {
			poolCount++
		}
	}
	return poolCount >= requiredPools
}

// DoesLeaseHaveNetworks returns true if a lease already has an associated network
func DoesLeaseHaveNetworks(lease *v1.Lease) bool {
	requiredNetworks := lease.Spec.Networks
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Network" {
			requiredNetworks--
		}
	}
	return requiredNetworks == 0
}

func GenerateEnvVars(lease *v1.Lease, pool *v1.Pool, network *v1.Network) error {
	var portgroup string
	for _, portgroup = range pool.Spec.Topology.Networks {
		if strings.Contains(portgroup, network.Spec.PortGroupName) {
			break
		}
	}
	tokens := strings.Split(portgroup, "/")
	if len(tokens) >= 3 {
		portgroup = tokens[len(tokens)-1]
	}
	inputs := struct {
		Server                string
		ComputeCluster        string
		ResourcePool          string
		VDatacenter           string
		Datastore             string
		PortGroup             string
		VlanId                string
		Gateway               string
		Nameserver            string
		IDatacenter           string
		PrimaryRouterHostname string
	}{
		Server:                pool.Spec.Server,
		ComputeCluster:        pool.Spec.Topology.ComputeCluster,
		ResourcePool:          pool.Spec.Topology.ResourcePool,
		VDatacenter:           pool.Spec.Topology.Datacenter,
		Datastore:             pool.Spec.Topology.Datastore,
		PortGroup:             portgroup,
		Gateway:               *network.Spec.Gateway,
		Nameserver:            *network.Spec.Gateway, // Default to Gateway for legacy usage.  We'll update below if nameservers set.
		VlanId:                network.Spec.VlanId,
		IDatacenter:           pool.Spec.IBMPoolSpec.Datacenter,
		PrimaryRouterHostname: network.Spec.PrimaryRouterHostname,
	}

	// If Nameserver set, then use it.
	if len(network.Spec.Nameservers) > 0 {
		inputs.Nameserver = network.Spec.Nameservers[0]
	}

	outBytes := new(bytes.Buffer)
	err := parsedTemplate.Execute(outBytes, inputs)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}
	envVarsString := outBytes.String()

	// Set the deprecated EnvVars field for backward compatibility
	lease.Status.EnvVars = envVarsString

	// Initialize EnvVarsMap if needed and add entry for this pool
	if lease.Status.EnvVarsMap == nil {
		lease.Status.EnvVarsMap = make(map[string]string)
	}
	lease.Status.EnvVarsMap[pool.Name] = envVarsString

	return nil
}

// GenerateEnvVarsForServer generates environment variables for a specific pool and network,
// returning the string without modifying the lease status
func GenerateEnvVarsForServer(pool *v1.Pool, network *v1.Network) (string, error) {
	var portgroup string
	for _, portgroup = range pool.Spec.Topology.Networks {
		if strings.Contains(portgroup, network.Spec.PortGroupName) {
			break
		}
	}
	tokens := strings.Split(portgroup, "/")
	if len(tokens) >= 3 {
		portgroup = tokens[len(tokens)-1]
	}
	inputs := struct {
		Server                string
		ComputeCluster        string
		ResourcePool          string
		VDatacenter           string
		Datastore             string
		PortGroup             string
		VlanId                string
		Gateway               string
		Nameserver            string
		IDatacenter           string
		PrimaryRouterHostname string
	}{
		Server:                pool.Spec.Server,
		ComputeCluster:        pool.Spec.Topology.ComputeCluster,
		ResourcePool:          pool.Spec.Topology.ResourcePool,
		VDatacenter:           pool.Spec.Topology.Datacenter,
		Datastore:             pool.Spec.Topology.Datastore,
		PortGroup:             portgroup,
		Gateway:               *network.Spec.Gateway,
		Nameserver:            *network.Spec.Gateway,
		VlanId:                network.Spec.VlanId,
		IDatacenter:           pool.Spec.IBMPoolSpec.Datacenter,
		PrimaryRouterHostname: network.Spec.PrimaryRouterHostname,
	}

	if len(network.Spec.Nameservers) > 0 {
		inputs.Nameserver = network.Spec.Nameservers[0]
	}

	outBytes := new(bytes.Buffer)
	err := parsedTemplate.Execute(outBytes, inputs)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}
	return outBytes.String(), nil
}
