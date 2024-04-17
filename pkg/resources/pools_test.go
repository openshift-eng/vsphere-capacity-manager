package resources

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/openshift-splat-team/vsphere-capacity-manager/data"
)

func TestPoolCalculation(t *testing.T) {
	t.Log("TestPoolCalculation")
	calculateResourceUsage()
	spew.Dump(Pools)

	_, err := getPoolsWithStrategy(&data.Resource{
		Spec: data.ResourceSpec{
			VCpus:    24,
			Memory:   96,
			Storage:  720,
			Networks: 1,
			VCenters: 1,
		},
	}, data.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
	if err != nil {
		t.Fatalf("error acquiring lease: %s", err)
	}

	leases, err := AcquireLease(&data.Resource{
		Spec: data.ResourceSpec{
			VCpus:    24,
			Memory:   96,
			Storage:  720,
			VCenters: 2,
		},
	})

	if err != nil {
		t.Fatalf("error acquiring lease: %s", err)
	}

	calculateResourceUsage()
	spew.Dump(leases)
}
