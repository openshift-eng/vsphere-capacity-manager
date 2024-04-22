package resources

import (
	"testing"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPoolsWithStrategy(t *testing.T) {
	t.Log("TestGetPoolsWithStrategy")

	// Create mock pools
	pools := v1.Pools{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pool1",
			},
			Spec: v1.PoolSpec{
				VCpus:    24,
				Memory:   96,
				Storage:  720,
				Networks: 1,
			},
			Status: v1.PoolStatus{
				VCpusAvailable:     24,
				MemoryAvailable:    96,
				DatastoreAvailable: 720,
				NetworkAvailable:   1,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pool2",
			},

			Spec: v1.PoolSpec{
				VCpus:    48,
				Memory:   192,
				Storage:  1440,
				Networks: 2,
			},

			Status: v1.PoolStatus{
				VCpusAvailable:     48,
				MemoryAvailable:    192,
				DatastoreAvailable: 1440,
				NetworkAvailable:   2,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "zonal-pool1",
			},

			Spec: v1.PoolSpec{
				VCpus:    48,
				Memory:   192,
				Storage:  1440,
				Networks: 2,
				Exclude:  true,
			},

			Status: v1.PoolStatus{
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
		expected v1.Pools
		resource *v1.ResourceRequest
	}{
		{
			name: "single vCenter, single network, sized for 3 control plane nodes and 3 computes, should pass",
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   1,
					},
				},
			},
			resource: &v1.ResourceRequest{
				Spec: v1.ResourceRequestSpec{
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
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   1,
					},
				},
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     48,
						MemoryAvailable:    192,
						DatastoreAvailable: 1440,
						NetworkAvailable:   2,
					},
				},
			},
			resource: &v1.ResourceRequest{
				Spec: v1.ResourceRequestSpec{
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
			pools, err := getPoolsWithStrategy(&tc.resource.Spec, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !arePoolsEqual(pools, tc.expected) {
				t.Errorf("unexpected fitting pools, got: %v, want: %v", pools, tc.expected)
			}
		})
	}
}

func arePoolsEqual(pools1, pools2 v1.Pools) bool {
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
