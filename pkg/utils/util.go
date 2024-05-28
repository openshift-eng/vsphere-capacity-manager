package utils

import (
	"bytes"
	"fmt"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"text/template"
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
		export dns_server="{{.Gateway}}"
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

	inputs := struct {
		Server                string
		ComputeCluster        string
		ResourcePool          string
		VDatacenter           string
		Datastore             string
		PortGroup             string
		VlanId                string
		Gateway               string
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
		VlanId:                network.Spec.VlanId,
		IDatacenter:           pool.Spec.IBMPoolSpec.Datacenter,
		PrimaryRouterHostname: network.Spec.PrimaryRouterHostname,
	}

	outBytes := new(bytes.Buffer)
	err := parsedTemplate.Execute(outBytes, inputs)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}
	lease.Status.EnvVars = outBytes.String()
	return nil
}
