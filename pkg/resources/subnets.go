package resources

import (
	"encoding/json"
	"log"
	"os"

	"github.com/openshift-splat-team/vsphere-capacity-manager/data"
)

func reconcileSubnets(pools data.Pools) {
	subnetsContent, err := os.ReadFile("/home/rvanderp/code/vsphere-capacity-manager/subnets.json")
	if err != nil {
		log.Fatalf("error reading subnets.json: %s", err)
	}

	var subnets data.Subnets
	err = json.Unmarshal(subnetsContent, &subnets)
	if err != nil {
		log.Fatalf("error unmarshalling subnets.json: %s", err)
	}

	for idx, pool := range pools {
		for _, datacenter := range subnets {
			for _, network := range datacenter {
				if pool.Spec.VCenter == network.Virtualcenter {
					pools[idx].Status.PortGroups = append(pool.Status.PortGroups, network)
				}
			}
		}
	}
}
