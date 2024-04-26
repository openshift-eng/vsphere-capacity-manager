package resources

import (
	"log"
	"testing"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPoolsWithStrategy(t *testing.T) {
	t.Log("TestGetPoolsWithStrategy")

	// Create mock pools
	pools := []*v1.Pool{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pool1",
			},
			Spec: v1.PoolSpec{
				VCpus:   24,
				Memory:  96,
				Storage: 720,
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
				VCpus:   48,
				Memory:  192,
				Storage: 1440,
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
				VCpus:   48,
				Memory:  192,
				Storage: 1440,
				Exclude: true,
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
	Pools := map[string]*v1.Pool{}
	for _, pool := range pools {
		Pools[pool.Name] = pool
	}

	testcases := []struct {
		name     string
		expected []*v1.Pool
		resource *v1.ResourceRequest
	}{
		{
			name: "single vCenter, single network, sized for 3 control plane nodes and 3 computes, should pass",
			expected: []*v1.Pool{
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
			expected: []*v1.Pool{
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

		})
	}
}

func TestGetFittingPools(t *testing.T) {
	t.Log("TestGetFittingPools")

	leases := []v1.Lease{
		{
			Spec: v1.LeaseSpec{
				VCpus:    24,
				Memory:   96,
				Storage:  720,
				Networks: 1,
			},
		},
		{
			Spec: v1.LeaseSpec{
				VCpus:    24,
				Memory:   96,
				Storage:  720,
				Networks: 1,
			},
		},
	}

	pools := []v1.Pool{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pool1",
			},
			Spec: v1.PoolSpec{
				VCpus:   24,
				Memory:  96,
				Storage: 720,
			},
			Status: v1.PoolStatus{
				VCpusAvailable:     24,
				MemoryAvailable:    96,
				DatastoreAvailable: 720,
				NetworkAvailable:   2,
				PortGroups: []v1.Network{
					{
						VifIpAddress: "192.168.0.1",
					},
					{
						VifIpAddress: "192.168.0.2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pool2",
			},
			Spec: v1.PoolSpec{
				VCpus:   48,
				Memory:  192,
				Storage: 1440,
			},
			Status: v1.PoolStatus{
				VCpusAvailable:     48,
				MemoryAvailable:    192,
				DatastoreAvailable: 1440,
				NetworkAvailable:   2,
				PortGroups: []v1.Network{
					{
						VifIpAddress: "192.168.0.1",
					},
					{
						VifIpAddress: "192.168.0.2",
					},
				},
			},
		},
	}

	expected := []*v1.Pool{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pool1",
			},
			Spec: v1.PoolSpec{
				VCpus:   24,
				Memory:  96,
				Storage: 720,
			},
			Status: v1.PoolStatus{
				VCpusAvailable:     0,
				MemoryAvailable:    0,
				DatastoreAvailable: 0,
				NetworkAvailable:   1,
				PortGroups: []v1.Network{
					{
						VifIpAddress: "192.168.0.1",
					},
					{
						VifIpAddress: "192.168.0.2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pool2",
			},
			Spec: v1.PoolSpec{
				VCpus:   24,
				Memory:  96,
				Storage: 720,
			},
			Status: v1.PoolStatus{
				VCpusAvailable:     24,
				MemoryAvailable:    96,
				DatastoreAvailable: 720,
				NetworkAvailable:   1,
				PortGroups: []v1.Network{
					{
						VifIpAddress: "192.168.0.1",
					},
					{
						VifIpAddress: "192.168.0.2",
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		leases   []v1.Lease
		pools    []v1.Pool
		expected []*v1.Pool
	}{
		{
			name:     "single vCenter, single network, sized for 3 control plane nodes and 3 computes, should pass",
			leases:   leases[:1],
			pools:    pools[:1],
			expected: expected[:1],
		},
		{
			name:     "dual leases, single network, sized for 3 control plane nodes and 3 computes, should pass",
			leases:   leases[:2],
			pools:    pools[:2],
			expected: expected[:2],
		},
	}
	for _, tc := range tests {
		fittedPools := []*v1.Pool{}
		t.Run(tc.name, func(t *testing.T) {
			for _, lease := range tc.leases {
				fittingPool, err := GetPoolWithStrategy(&lease, pools, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				fittingPool.Status.VCpusAvailable -= lease.Spec.VCpus
				fittingPool.Status.MemoryAvailable -= lease.Spec.Memory
				fittingPool.Status.DatastoreAvailable -= lease.Spec.Storage
				fittedPools = append(fittedPools, fittingPool)
			}
			if !arePoolsEqual(fittedPools, tc.expected) {
				t.Errorf("unexpected fitting pools, got: %v, want: %v", fittedPools, expected)
			}

		})
	}

}

// BEGIN: arePoolsEqual
func arePoolsEqual(pools1, pools2 []*v1.Pool) bool {
	if len(pools1) != len(pools2) {
		return false
	}
	for _, pool1 := range pools1 {
		log.Printf("pool1: vCPU: %d memory: %d storage: %d", pool1.Status.VCpusAvailable, pool1.Status.MemoryAvailable, pool1.Status.DatastoreAvailable)
		hasMatch := false
		for _, pool2 := range pools2 {
			log.Printf("pool2: vCPU: %d memory: %d storage: %d", pool2.Status.VCpusAvailable, pool2.Status.MemoryAvailable, pool2.Status.DatastoreAvailable)
			if pool1.Status.VCpusAvailable == pool2.Status.VCpusAvailable &&
				pool1.Status.MemoryAvailable == pool2.Status.MemoryAvailable &&
				pool1.Status.DatastoreAvailable == pool2.Status.DatastoreAvailable {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			return false
		}
	}
	return true
}

// END: arePoolsEqual
