package resources

import (
	"testing"

	"github.com/openshift-splat-team/vsphere-capacity-manager/data"
)

func TestGetPoolsWithStrategy(t *testing.T) {
	t.Log("TestGetPoolsWithStrategy")

	// Create mock pools
	pools := data.Pools{
		{
			Spec: data.PoolSpec{
				ResourceSpec: data.ResourceSpec{
					VCpus:    24,
					Memory:   96,
					Storage:  720,
					Networks: 1,
				},
			},
			Status: data.PoolStatus{
				VCpusAvailable:     24,
				MemoryAvailable:    96,
				DatastoreAvailable: 720,
				NetworkAvailable:   1,
			},
		},
		{
			Spec: data.PoolSpec{
				ResourceSpec: data.ResourceSpec{
					VCpus:    48,
					Memory:   192,
					Storage:  1440,
					Networks: 2,
				},
			},

			Status: data.PoolStatus{
				VCpusAvailable:     48,
				MemoryAvailable:    192,
				DatastoreAvailable: 1440,
				NetworkAvailable:   2,
			},
		},
	}

	// Set the global Pools variable to the mock pools
	Pools = pools

	testcases := []struct {
		name     string
		expected data.Pools
		resource *data.Resource
	}{
		{
			name: "single vCenter, single network, sized for 3 control plane nodes and 3 computes, should pass",
			expected: data.Pools{
				{
					Status: data.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   1,
					},
				},
			},
			resource: &data.Resource{
				Spec: data.ResourceSpec{
					VCpus:    24,
					Memory:   96,
					Storage:  720,
					Networks: 1,
					VCenters: 1,
				},
			},
		},
		{
			name: "single vCenter, single network, sized for 3 control plane nodes and 3 computes, should pass",
			expected: data.Pools{
				{
					Status: data.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   1,
					},
				},
				{
					Status: data.PoolStatus{
						VCpusAvailable:     48,
						MemoryAvailable:    192,
						DatastoreAvailable: 1440,
						NetworkAvailable:   2,
					},
				},
			},
			resource: &data.Resource{
				Spec: data.ResourceSpec{
					VCpus:    24,
					Memory:   96,
					Storage:  720,
					Networks: 1,
					VCenters: 2,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			pools, err := getPoolsWithStrategy(tc.resource, data.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !arePoolsEqual(pools, tc.expected) {
				t.Errorf("unexpected fitting pools, got: %v, want: %v", pools, tc.expected)
			}
		})
	}
}

func arePoolsEqual(pools1, pools2 data.Pools) bool {
	if len(pools1) != len(pools2) {
		return false
	}
	for i := range pools1 {
		if pools1[i].Status.VCpusAvailable != pools2[i].Status.VCpusAvailable ||
			pools1[i].Status.MemoryAvailable != pools2[i].Status.MemoryAvailable ||
			pools1[i].Status.DatastoreAvailable != pools2[i].Status.DatastoreAvailable ||
			pools1[i].Status.NetworkAvailable != pools2[i].Status.NetworkAvailable {
			return false
		}
	}
	return true
}
