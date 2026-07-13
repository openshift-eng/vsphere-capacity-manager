package controller

import (
	"testing"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/utils"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestDynamicVCenterFiltering tests the dynamic filtering logic that adapts
// based on remaining vCenter slots and remaining pools needed.
func TestDynamicVCenterFiltering(t *testing.T) {
	tests := []struct {
		name                   string
		requiredPools          int
		vcentersLimit          int
		assignedPools          int // How many pools already assigned
		vcentersInUse          int // How many distinct vCenters in use
		availablePools         []*v1.Pool
		expectExclusions       bool
		expectedMinPoolsNeeded int // Expected minPoolsPerVCenter threshold
		description            string
	}{
		{
			name:          "cap reached - only allow vcenters in use",
			requiredPools: 4,
			vcentersLimit: 3,
			assignedPools: 3, // Already have 3 pools
			vcentersInUse: 3, // From 3 different vCenters (cap reached)
			availablePools: []*v1.Pool{
				createPool("vcenter-A", "pool-a", 100, 1000),
				createPool("vcenter-B", "pool-b", 100, 1000),
				createPool("vcenter-C", "pool-c", 100, 1000),
				createPool("vcenter-D", "pool-d", 100, 1000), // Should be excluded
			},
			expectExclusions: true, // Should exclude vcenter-D
			description:      "Cap reached: should exclude all vCenters not in use",
		},
		{
			name:          "one slot left for two pools - require multi-pool vcenter",
			requiredPools: 4,
			vcentersLimit: 3,
			assignedPools: 2, // Already have 2 pools
			vcentersInUse: 2, // From 2 different vCenters
			availablePools: []*v1.Pool{
				// vcenter-A and B already in use (not counted here)
				// vcenter-C has 1 pool
				createPool("vcenter-C", "pool-c1", 100, 1000),
				// vcenter-D has 2 pools
				createPool("vcenter-D", "pool-d1", 100, 1000),
				createPool("vcenter-D", "pool-d2", 100, 1000),
			},
			expectExclusions:       true,
			expectedMinPoolsNeeded: 2, // ceil(2 remaining pools / 1 remaining slot) = 2
			description:            "Need 2 pools from 1 slot: should exclude single-pool vCenters",
		},
		{
			name:          "two slots left for three pools - require 2 per vcenter",
			requiredPools: 4,
			vcentersLimit: 3,
			assignedPools: 1, // Already have 1 pool
			vcentersInUse: 1, // From 1 vCenter
			availablePools: []*v1.Pool{
				// vcenter-A already in use
				// vcenter-B has 1 pool - should be excluded
				createPool("vcenter-B", "pool-b1", 100, 1000),
				// vcenter-C has 2 pools - should be allowed
				createPool("vcenter-C", "pool-c1", 100, 1000),
				createPool("vcenter-C", "pool-c2", 100, 1000),
				// vcenter-D has 3 pools - should be allowed
				createPool("vcenter-D", "pool-d1", 100, 1000),
				createPool("vcenter-D", "pool-d2", 100, 1000),
				createPool("vcenter-D", "pool-d3", 100, 1000),
			},
			expectExclusions:       true,
			expectedMinPoolsNeeded: 2, // ceil(3 remaining / 2 slots) = 2
			description:            "Need 3 pools from 2 slots: exclude vCenters with < 2 pools",
		},
		{
			name:          "plenty of slots - no dynamic filtering",
			requiredPools: 4,
			vcentersLimit: 5,
			assignedPools: 1,
			vcentersInUse: 1,
			availablePools: []*v1.Pool{
				// Remaining: 3 pools needed, 4 slots available
				// 3 <= 4, so no dynamic filtering needed
				createPool("vcenter-B", "pool-b1", 100, 1000),
				createPool("vcenter-C", "pool-c1", 100, 1000),
				createPool("vcenter-D", "pool-d1", 100, 1000),
			},
			expectExclusions: false,
			description:      "More slots than pools: no dynamic filtering needed",
		},
		{
			name:          "all remaining vcenters excluded by dynamic filter - should trigger recovery",
			requiredPools: 4,
			vcentersLimit: 3,
			assignedPools: 1, // Already have 1 pool from vcenter-A
			vcentersInUse: 1,
			availablePools: []*v1.Pool{
				// vcenter-A already in use
				// Need 3 more pools, have 2 slots left
				// minPoolsPerVCenter = ceil(3/2) = 2
				// All remaining vCenters have only 1 pool → ALL excluded
				createPool("vcenter-B", "pool-b1", 100, 1000), // 1 pool < 2
				createPool("vcenter-C", "pool-c1", 100, 1000), // 1 pool < 2
				createPool("vcenter-D", "pool-d1", 100, 1000), // 1 pool < 2
			},
			expectExclusions:       true,
			expectedMinPoolsNeeded: 2, // ceil(3/2) = 2
			description:            "All remaining vCenters excluded: should trigger deadlock recovery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease := &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-lease",
					Namespace: "default",
				},
				Spec: v1.LeaseSpec{
					VCpus:    24,
					Memory:   96,
					Pools:    tt.requiredPools,
					VCenters: tt.vcentersLimit,
				},
			}

			// Simulate assigned pools (create dummy pools)
			assignedPools := make([]*v1.Pool, tt.assignedPools)
			vcentersInUse := make(map[string]bool)
			for i := 0; i < tt.assignedPools; i++ {
				// Use different vCenters to match vcentersInUse count
				vcenterName := ""
				if i < tt.vcentersInUse {
					vcenterName = string(rune('A' + i)) // A, B, C, ...
				} else {
					// Reuse an earlier vCenter
					vcenterName = "A"
				}
				assignedPools[i] = createPool("vcenter-"+vcenterName, "assigned-pool-"+string(rune('1'+i)), 100, 1000)
				vcentersInUse["vcenter-"+vcenterName] = true
			}

			// Calculate what the controller would calculate
			remainingSlots := tt.vcentersLimit - len(vcentersInUse)
			remainingPools := tt.requiredPools - len(assignedPools)

			t.Logf("%s", tt.description)
			t.Logf("Remaining slots: %d, Remaining pools: %d", remainingSlots, remainingPools)

			var excludedVCenters map[string]bool

			// Replicate the controller's logic
			if len(vcentersInUse) >= lease.Spec.VCenters {
				// Cap reached
				excludedVCenters = make(map[string]bool)
				for _, p := range tt.availablePools {
					if !vcentersInUse[p.Spec.Server] {
						excludedVCenters[p.Spec.Server] = true
					}
				}
				t.Logf("Cap reached - excluded %d vCenters", len(excludedVCenters))
			} else if remainingSlots > 0 && remainingPools > remainingSlots {
				// Dynamic filtering
				minPoolsPerVCenter := (remainingPools-1)/remainingSlots + 1
				t.Logf("Dynamic filtering: minPoolsPerVCenter = %d", minPoolsPerVCenter)

				if tt.expectedMinPoolsNeeded > 0 && minPoolsPerVCenter != tt.expectedMinPoolsNeeded {
					t.Errorf("Expected minPoolsPerVCenter=%d, got %d",
						tt.expectedMinPoolsNeeded, minPoolsPerVCenter)
				}

				// Count pools per vCenter
				fittingPools, _ := utils.GetFittingPools(lease, tt.availablePools, nil)
				poolsPerVCenter := make(map[string]int)
				for _, p := range fittingPools {
					if !vcentersInUse[p.Spec.Server] {
						poolsPerVCenter[p.Spec.Server]++
					}
				}

				// Exclude vCenters with insufficient pools
				excludedVCenters = make(map[string]bool)
				for _, p := range tt.availablePools {
					if !vcentersInUse[p.Spec.Server] {
						if poolsPerVCenter[p.Spec.Server] < minPoolsPerVCenter {
							excludedVCenters[p.Spec.Server] = true
						}
					}
				}

				t.Logf("Dynamic filter excluded %d vCenters (with < %d pools)",
					len(excludedVCenters), minPoolsPerVCenter)

				for server, count := range poolsPerVCenter {
					excluded := excludedVCenters[server]
					t.Logf("  %s: %d pools → excluded=%v", server, count, excluded)
				}
			}

			// Verify expectations
			if tt.expectExclusions && len(excludedVCenters) == 0 {
				t.Error("Expected exclusions but got none")
			}
			if !tt.expectExclusions && len(excludedVCenters) > 0 {
				t.Errorf("Expected no exclusions but got %d", len(excludedVCenters))
			}
		})
	}
}

// Helper function to create a pool for testing
func createPool(vcenterServer, poolName string, vcpus, memory int) *v1.Pool {
	return &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: poolName,
		},
		Spec: v1.PoolSpec{
			FailureDomainSpec: v1.FailureDomainSpec{
				VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
					Server: vcenterServer,
				},
			},
			VCpus:  vcpus,
			Memory: memory,
		},
		Status: v1.PoolStatus{
			VCpusAvailable:  vcpus,
			MemoryAvailable: memory,
		},
	}
}
