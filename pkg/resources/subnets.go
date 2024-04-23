package resources

import (
	"encoding/json"
	"log"
	"os"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

// ReconcileSubnets reconciles the subnets with the pools
func ReconcileSubnets(pools v1.Pools) {
	subnetsContent, err := os.ReadFile("/home/rvanderp/code/vsphere-capacity-manager/subnets.json")
	if err != nil {
		log.Fatalf("error reading subnets.json: %s", err)
	}

	var subnets v1.Subnets
	err = json.Unmarshal(subnetsContent, &subnets)
	if err != nil {
		log.Fatalf("error unmarshalling subnets.json: %s", err)
	}

	for idx, pool := range pools {
		pools[idx].Status.PortGroups = []v1.Network{}
		for _, datacenter := range subnets {
			for _, network := range datacenter {
				if pool.Spec.Server == network.Virtualcenter {
					pools[idx].Status.PortGroups = append(pool.Status.PortGroups, network)
				}
			}
		}
	}
}
