package resources

import (
	"fmt"
	"log"
	"testing"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func compareSlices(slice1 []v1.PoolStatus, slice2 []v1.PoolStatus) bool {
	if len(slice1) != len(slice2) {
		log.Printf("slice1 and slice2 are not the same length")
		return false
	}

	for i := range slice1 {
		if slice1[i].VCpusAvailable != slice2[i].VCpusAvailable ||
			slice1[i].MemoryAvailable != slice2[i].MemoryAvailable ||
			slice1[i].DatastoreAvailable != slice2[i].DatastoreAvailable ||
			slice1[i].NetworkAvailable != slice2[i].NetworkAvailable ||
			len(slice1[i].PortGroups) != len(slice2[i].PortGroups) {
			log.Printf("slice1 fields and slice2 fields are not the same")
			log.Printf("VCpusAvailable: %d %d", slice1[i].VCpusAvailable, slice2[i].VCpusAvailable)
			log.Printf("MemoryAvailable: %d %d", slice1[i].MemoryAvailable, slice2[i].MemoryAvailable)
			log.Printf("DatastoreAvailable: %d %d", slice1[i].DatastoreAvailable, slice2[i].DatastoreAvailable)
			log.Printf("NetworkAvailable: %d %d", slice1[i].NetworkAvailable, slice2[i].NetworkAvailable)
			log.Printf("PortGroups length: %d %d", len(slice1[i].PortGroups), len(slice2[i].PortGroups))
			return false
		}

		for _, pg1 := range slice1[i].PortGroups {
			match := false
			for _, pg2 := range slice2[i].PortGroups {
				if pg1.Network == pg2.Network {
					match = true
					break
				}
			}
			if match == false {
				log.Printf("unable to find port group %s in slice2", pg1.Network)
				return false
			}
		}
	}

	return true
}

func constructTestPools(num int) v1.Pools {
	pools := make(v1.Pools, num)
	for idx := 0; idx < num; idx++ {
		pools[idx] = &v1.Pool{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("pool_%d", idx),
			},
			Spec: v1.PoolSpec{
				ResourceRequestSpec: v1.ResourceRequestSpec{
					VCpus:   24 * (idx + 1),
					Memory:  96 * (idx + 1),
					Storage: 720 * (idx + 1),
				},
			},
			Status: v1.PoolStatus{
				VCpusAvailable:     24 * (idx + 1),
				MemoryAvailable:    96 * (idx + 1),
				DatastoreAvailable: 720 * (idx + 1),
				PortGroups: []v1.Network{
					{
						Network: "network1",
					},
					{
						Network: "network2",
					},
					{
						Network: "network3",
					},
				},
			},
		}
	}
	return pools
}

func TestAcquireLease(t *testing.T) {
	t.Log("TestAcquireLease")

	testcases := []struct {
		name     string
		expected v1.Pools
		resource *v1.ResourceRequest
		error    string
	}{
		{
			name: "single vCenter, single network, sized for 3 control plane nodes and 3 computes, should pass",
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     0,
						MemoryAvailable:    0,
						DatastoreAvailable: 0,
						NetworkAvailable:   2,
						PortGroups: []v1.Network{
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     48,
						MemoryAvailable:    192,
						DatastoreAvailable: 1440,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
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
			name:  "single vCenter, single network, sized for 12 control plane nodes and 12 computes, should fail",
			error: `error acquiring lease: no pools with enough resources`,
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     48,
						MemoryAvailable:    192,
						DatastoreAvailable: 1440,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
			},
			resource: &v1.ResourceRequest{
				Spec: v1.ResourceRequestSpec{
					VCpus:    4 * 24,
					Memory:   16 * 24,
					Storage:  120 * 24,
					Networks: 24,
					VCenters: 1,
				},
			},
		},
		{
			name: "dual vCenters, single network, sized for 3 control plane nodes and 3 computes, 2 pools available, should pass",
			resource: &v1.ResourceRequest{
				Spec: v1.ResourceRequestSpec{
					VCpus:    24,
					Memory:   96,
					Storage:  720,
					Networks: 1,
					VCenters: 2,
				},
			},
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     0,
						MemoryAvailable:    0,
						DatastoreAvailable: 0,
						NetworkAvailable:   2,
						PortGroups: []v1.Network{
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   2,
						PortGroups: []v1.Network{
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
			},
		},
		{
			name: "three vCenters, single network, sized for 3 control plane nodes and 3 computes, 2 pools available, should fail",
			resource: &v1.ResourceRequest{
				Spec: v1.ResourceRequestSpec{
					VCpus:    24,
					Memory:   96,
					Storage:  720,
					Networks: 1,
					VCenters: 3,
				},
			},
			error: `error acquiring lease: required number of vCenters exceeds the number of fitting pools`,
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     48,
						MemoryAvailable:    192,
						DatastoreAvailable: 1440,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			Pools = constructTestPools(2)

			leases, err := AcquireLease(*tc.resource)
			if err != nil {
				if len(tc.error) > 0 {
					if err.Error() == tc.error {
						return
					} else {
						t.Errorf("expected error: %v but got %v", tc.error, err)
					}
				}
				t.Errorf("unexpected error: %v", err)
			}
			calculateResourceUsage()
			poolStatus := make([]v1.PoolStatus, len(Pools))
			for i := range Pools {
				poolStatus[i] = Pools[i].Status
			}
			expectedPoolStatus := make([]v1.PoolStatus, len(tc.expected))
			for i := range tc.expected {
				expectedPoolStatus[i] = tc.expected[i].Status
			}
			if !compareSlices(poolStatus, expectedPoolStatus) {
				t.Fatalf("unexpected pool status")
			}

			// check that leases have been granted their requested resources
			for _, lease := range *leases {
				if lease.Spec.ResourceRequestSpec.VCenters != tc.resource.Spec.VCenters ||
					lease.Spec.ResourceRequestSpec.VCpus != tc.resource.Spec.VCpus ||
					lease.Spec.ResourceRequestSpec.Memory != tc.resource.Spec.Memory ||
					lease.Spec.ResourceRequestSpec.Storage != tc.resource.Spec.Storage ||
					lease.Spec.ResourceRequestSpec.Networks != tc.resource.Spec.Networks ||
					len(lease.Status.PortGroups) != tc.resource.Spec.Networks {
					t.Errorf("lease resource spec does not match the requested resource spec")
				}
			}
		})
	}
}

func TestReleaseLease(t *testing.T) {
	t.Log("TestReleaeLease")

	testcases := []struct {
		name     string
		expected v1.Pools
		resource *v1.ResourceRequest
		error    string
	}{
		{
			name: "single vCenter, single network, sized for 3 control plane nodes and 3 computes, should pass",
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     48,
						MemoryAvailable:    192,
						DatastoreAvailable: 1440,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
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
			name: "dual vCenters, single network, sized for 3 control plane nodes and 3 computes, 2 pools available, should pass",
			resource: &v1.ResourceRequest{
				Spec: v1.ResourceRequestSpec{
					VCpus:    24,
					Memory:   96,
					Storage:  720,
					Networks: 1,
					VCenters: 2,
				},
			},
			expected: v1.Pools{
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     24,
						MemoryAvailable:    96,
						DatastoreAvailable: 720,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
				{
					Status: v1.PoolStatus{
						VCpusAvailable:     48,
						MemoryAvailable:    192,
						DatastoreAvailable: 1440,
						NetworkAvailable:   3,
						PortGroups: []v1.Network{
							{
								Network: "network1",
							},
							{
								Network: "network2",
							},
							{
								Network: "network3",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			Pools = constructTestPools(2)

			leases, err := AcquireLease(*tc.resource)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			err = ReleaseLease(leases)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			poolStatus := make([]v1.PoolStatus, len(Pools))
			for i := range Pools {
				poolStatus[i] = Pools[i].Status
			}
			expectedPoolStatus := make([]v1.PoolStatus, len(tc.expected))
			for i := range tc.expected {
				expectedPoolStatus[i] = tc.expected[i].Status
			}
			if !compareSlices(poolStatus, expectedPoolStatus) {
				t.Fatalf("unexpected pool status")
			}
		})
	}
}
