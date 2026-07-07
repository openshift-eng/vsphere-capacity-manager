package utils

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

func TestTolerationMatchesTaint(t *testing.T) {
	tests := []struct {
		name       string
		toleration v1.Toleration
		taint      v1.Taint
		expected   bool
	}{
		{
			name: "exact match with Equal operator",
			toleration: v1.Toleration{
				Key:      "dedicated",
				Operator: v1.TolerationOpEqual,
				Value:    "gpu",
				Effect:   "NoSchedule",
			},
			taint: v1.Taint{
				Key:    "dedicated",
				Value:  "gpu",
				Effect: v1.TaintEffectNoSchedule,
			},
			expected: true,
		},
		{
			name: "Equal operator key mismatch",
			toleration: v1.Toleration{
				Key:      "dedicated",
				Operator: v1.TolerationOpEqual,
				Value:    "gpu",
			},
			taint: v1.Taint{
				Key:    "special",
				Value:  "gpu",
				Effect: v1.TaintEffectNoSchedule,
			},
			expected: false,
		},
		{
			name: "Equal operator value mismatch",
			toleration: v1.Toleration{
				Key:      "dedicated",
				Operator: v1.TolerationOpEqual,
				Value:    "gpu",
			},
			taint: v1.Taint{
				Key:    "dedicated",
				Value:  "cpu",
				Effect: v1.TaintEffectNoSchedule,
			},
			expected: false,
		},
		{
			name: "Exists operator matches key",
			toleration: v1.Toleration{
				Key:      "dedicated",
				Operator: v1.TolerationOpExists,
			},
			taint: v1.Taint{
				Key:    "dedicated",
				Value:  "gpu",
				Effect: v1.TaintEffectNoSchedule,
			},
			expected: true,
		},
		{
			name: "Exists operator with empty key matches all taints",
			toleration: v1.Toleration{
				Key:      "",
				Operator: v1.TolerationOpExists,
			},
			taint: v1.Taint{
				Key:    "any-taint",
				Value:  "any-value",
				Effect: v1.TaintEffectNoSchedule,
			},
			expected: true,
		},
		{
			name: "Exists operator key mismatch",
			toleration: v1.Toleration{
				Key:      "dedicated",
				Operator: v1.TolerationOpExists,
			},
			taint: v1.Taint{
				Key:    "special",
				Value:  "gpu",
				Effect: v1.TaintEffectNoSchedule,
			},
			expected: false,
		},
		{
			name: "effect mismatch",
			toleration: v1.Toleration{
				Key:      "dedicated",
				Operator: v1.TolerationOpEqual,
				Value:    "gpu",
				Effect:   "NoSchedule",
			},
			taint: v1.Taint{
				Key:    "dedicated",
				Value:  "gpu",
				Effect: v1.TaintEffectPreferNoSchedule,
			},
			expected: false,
		},
		{
			name: "toleration with empty effect matches any effect",
			toleration: v1.Toleration{
				Key:      "dedicated",
				Operator: v1.TolerationOpEqual,
				Value:    "gpu",
				Effect:   "",
			},
			taint: v1.Taint{
				Key:    "dedicated",
				Value:  "gpu",
				Effect: v1.TaintEffectPreferNoSchedule,
			},
			expected: true,
		},
		{
			name: "default Equal operator without explicit operator",
			toleration: v1.Toleration{
				Key:   "dedicated",
				Value: "gpu",
			},
			taint: v1.Taint{
				Key:    "dedicated",
				Value:  "gpu",
				Effect: v1.TaintEffectNoSchedule,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tolerationMatchesTaint(&tt.toleration, &tt.taint)
			if result != tt.expected {
				t.Errorf("tolerationMatchesTaint() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLeaseToleratesPoolTaints(t *testing.T) {
	tests := []struct {
		name     string
		lease    *v1.Lease
		pool     *v1.Pool
		expected bool
	}{
		{
			name: "pool with no taints",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					Tolerations: []v1.Toleration{},
				},
			},
			pool: &v1.Pool{
				Spec: v1.PoolSpec{
					Taints: []v1.Taint{},
				},
			},
			expected: true,
		},
		{
			name: "lease tolerates all pool taints",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					Tolerations: []v1.Toleration{
						{
							Key:      "dedicated",
							Operator: v1.TolerationOpEqual,
							Value:    "gpu",
							Effect:   "NoSchedule",
						},
					},
				},
			},
			pool: &v1.Pool{
				Spec: v1.PoolSpec{
					Taints: []v1.Taint{
						{
							Key:    "dedicated",
							Value:  "gpu",
							Effect: v1.TaintEffectNoSchedule,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "lease does not tolerate pool taints",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					Tolerations: []v1.Toleration{
						{
							Key:      "other",
							Operator: v1.TolerationOpEqual,
							Value:    "value",
						},
					},
				},
			},
			pool: &v1.Pool{
				Spec: v1.PoolSpec{
					Taints: []v1.Taint{
						{
							Key:    "dedicated",
							Value:  "gpu",
							Effect: v1.TaintEffectNoSchedule,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "lease tolerates multiple pool taints",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					Tolerations: []v1.Toleration{
						{
							Key:      "dedicated",
							Operator: v1.TolerationOpEqual,
							Value:    "gpu",
						},
						{
							Key:      "special",
							Operator: v1.TolerationOpExists,
						},
					},
				},
			},
			pool: &v1.Pool{
				Spec: v1.PoolSpec{
					Taints: []v1.Taint{
						{
							Key:    "dedicated",
							Value:  "gpu",
							Effect: v1.TaintEffectNoSchedule,
						},
						{
							Key:    "special",
							Value:  "true",
							Effect: v1.TaintEffectPreferNoSchedule,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "lease tolerates some but not all pool taints",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					Tolerations: []v1.Toleration{
						{
							Key:      "dedicated",
							Operator: v1.TolerationOpEqual,
							Value:    "gpu",
						},
					},
				},
			},
			pool: &v1.Pool{
				Spec: v1.PoolSpec{
					Taints: []v1.Taint{
						{
							Key:    "dedicated",
							Value:  "gpu",
							Effect: v1.TaintEffectNoSchedule,
						},
						{
							Key:    "special",
							Value:  "true",
							Effect: v1.TaintEffectNoSchedule,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "wildcard toleration matches all taints",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					Tolerations: []v1.Toleration{
						{
							Key:      "",
							Operator: v1.TolerationOpExists,
						},
					},
				},
			},
			pool: &v1.Pool{
				Spec: v1.PoolSpec{
					Taints: []v1.Taint{
						{
							Key:    "taint1",
							Value:  "value1",
							Effect: v1.TaintEffectNoSchedule,
						},
						{
							Key:    "taint2",
							Value:  "value2",
							Effect: v1.TaintEffectPreferNoSchedule,
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := leaseToleratesPoolTaints(tt.lease, tt.pool)
			if result != tt.expected {
				t.Errorf("leaseToleratesPoolTaints() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestPoolMatchesSelector(t *testing.T) {
	tests := []struct {
		name     string
		lease    *v1.Lease
		pool     *v1.Pool
		expected bool
	}{
		{
			name: "no selector specified",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					PoolSelector: map[string]string{},
				},
			},
			pool: &v1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"region": "us-west",
					},
				},
			},
			expected: true,
		},
		{
			name: "selector matches pool labels",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					PoolSelector: map[string]string{
						"region": "us-west",
						"tier":   "gpu",
					},
				},
			},
			pool: &v1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"region": "us-west",
						"tier":   "gpu",
						"zone":   "a",
					},
				},
			},
			expected: true,
		},
		{
			name: "selector key missing in pool labels",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					PoolSelector: map[string]string{
						"region": "us-west",
					},
				},
			},
			pool: &v1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"tier": "gpu",
					},
				},
			},
			expected: false,
		},
		{
			name: "selector value mismatch",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					PoolSelector: map[string]string{
						"region": "us-west",
					},
				},
			},
			pool: &v1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"region": "us-east",
					},
				},
			},
			expected: false,
		},
		{
			name: "pool has no labels",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					PoolSelector: map[string]string{
						"region": "us-west",
					},
				},
			},
			pool: &v1.Pool{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := poolMatchesSelector(tt.lease, tt.pool)
			if result != tt.expected {
				t.Errorf("poolMatchesSelector() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetFittingPools(t *testing.T) {
	tests := []struct {
		name               string
		lease              *v1.Lease
		pools              []*v1.Pool
		excludedVCenters   map[string]bool
		expectedFittingLen int
		expectedRejections map[string]string
	}{
		{
			name: "pool selector filters out non-matching pools",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
					PoolSelector: map[string]string{
						"region": "us-west",
					},
				},
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool1",
						Labels: map[string]string{
							"region": "us-west",
						},
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool2",
						Labels: map[string]string{
							"region": "us-east",
						},
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			expectedFittingLen: 1,
			expectedRejections: map[string]string{
				"pool2": PoolLabelMismatch,
			},
		},
		{
			name: "taint toleration filters pools",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
					Tolerations: []v1.Toleration{
						{
							Key:      "dedicated",
							Operator: v1.TolerationOpEqual,
							Value:    "gpu",
						},
					},
				},
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool1",
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
						Taints: []v1.Taint{
							{
								Key:    "dedicated",
								Value:  "gpu",
								Effect: v1.TaintEffectNoSchedule,
							},
						},
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool2",
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
						Taints: []v1.Taint{
							{
								Key:    "special",
								Value:  "true",
								Effect: v1.TaintEffectNoSchedule,
							},
						},
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			expectedFittingLen: 1,
			expectedRejections: map[string]string{
				"pool2": PoolTaintNotTolerated,
			},
		},
		{
			name: "combined selector and taint filtering",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
					PoolSelector: map[string]string{
						"region": "us-west",
					},
					Tolerations: []v1.Toleration{
						{
							Key:      "dedicated",
							Operator: v1.TolerationOpEqual,
							Value:    "gpu",
						},
					},
				},
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool1-matching",
						Labels: map[string]string{
							"region": "us-west",
						},
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
						Taints: []v1.Taint{
							{
								Key:    "dedicated",
								Value:  "gpu",
								Effect: v1.TaintEffectNoSchedule,
							},
						},
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool2-wrong-region",
						Labels: map[string]string{
							"region": "us-east",
						},
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
						Taints: []v1.Taint{
							{
								Key:    "dedicated",
								Value:  "gpu",
								Effect: v1.TaintEffectNoSchedule,
							},
						},
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool3-wrong-taint",
						Labels: map[string]string{
							"region": "us-west",
						},
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
						Taints: []v1.Taint{
							{
								Key:    "special",
								Value:  "true",
								Effect: v1.TaintEffectNoSchedule,
							},
						},
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			expectedFittingLen: 1,
			expectedRejections: map[string]string{
				"pool2-wrong-region": PoolLabelMismatch,
				"pool3-wrong-taint":  PoolTaintNotTolerated,
			},
		},
		{
			name: "no pools match due to insufficient resources",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  100,
					Memory: 200,
				},
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool1",
					},
					Spec: v1.PoolSpec{
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			expectedFittingLen: 0,
			expectedRejections: map[string]string{
				"pool1": PoolInsufficientVCPU,
			},
		},
		{
			name: "excludedVCenters filters pools on capped vcenters",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
				},
			},
			excludedVCenters: map[string]bool{
				"vcenter1.example.com": true,
				"vcenter2.example.com": true,
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-vc1",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-vc2",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-vc3",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter3.example.com",
							},
						},
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			// pool-vc3 is the only pool on a non-excluded vcenter
			expectedFittingLen: 1,
			expectedRejections: map[string]string{
				"pool-vc1": PoolVCenterLimitReached,
				"pool-vc2": PoolVCenterLimitReached,
			},
		},
		{
			name: "nil excludedVCenters applies no vcenter constraint",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
				},
			},
			excludedVCenters: nil,
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-vc1",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-vc2",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			// Both pools are fitting since no vcenter constraint is applied
			expectedFittingLen: 2,
			expectedRejections: map[string]string{},
		},
		{
			name: "pool on non-excluded vcenter passes vcenter check",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
				},
			},
			excludedVCenters: map[string]bool{
				"vcenter1.example.com": true,
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-excluded",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-allowed",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus: 100,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			expectedFittingLen: 1,
			expectedRejections: map[string]string{
				"pool-excluded": PoolVCenterLimitReached,
			},
		},
		{
			name: "vcenter check reports correct reason when pool has NoSchedule",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
				},
			},
			excludedVCenters: map[string]bool{
				"vcenter1.example.com": true,
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-noschedule",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:      100,
						NoSchedule: true, // This should be reported instead of vcenter limit
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			expectedFittingLen: 0,
			expectedRejections: map[string]string{
				// Should report NoSchedule, not PoolVCenterLimitReached
				"pool-noschedule": PoolNotSchedulable,
			},
		},
		{
			name: "vcenter check reports correct reason when pool is excluded",
			lease: &v1.Lease{
				Spec: v1.LeaseSpec{
					VCpus:  16,
					Memory: 32,
				},
			},
			excludedVCenters: map[string]bool{
				"vcenter1.example.com": true,
			},
			pools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pool-excluded",
					},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:   100,
						Exclude: true, // This should be reported instead of vcenter limit
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  50,
						MemoryAvailable: 100,
					},
				},
			},
			expectedFittingLen: 0,
			expectedRejections: map[string]string{
				// Should report Exclude, not PoolVCenterLimitReached
				"pool-excluded": PoolExcluded,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fittingPools, poolResults := GetFittingPools(tt.lease, tt.pools, tt.excludedVCenters)

			if len(fittingPools) != tt.expectedFittingLen {
				t.Errorf("GetFittingPools() returned %d fitting pools, expected %d",
					len(fittingPools), tt.expectedFittingLen)
			}

			for poolName, expectedReason := range tt.expectedRejections {
				found := false
				for _, result := range poolResults {
					if result.Pool.Name == poolName {
						found = true
						if result.MatchResults != expectedReason {
							t.Errorf("Pool %s has reason '%s', expected '%s'",
								poolName, result.MatchResults, expectedReason)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected rejection for pool %s not found in results", poolName)
				}
			}
		})
	}
}

func TestGetVCentersInUse(t *testing.T) {
	tests := []struct {
		name          string
		assignedPools []*v1.Pool
		expected      map[string]bool
	}{
		{
			name:          "empty assigned pools returns empty map",
			assignedPools: []*v1.Pool{},
			expected:      map[string]bool{},
		},
		{
			name: "single pool returns its vcenter server",
			assignedPools: []*v1.Pool{
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
					},
				},
			},
			expected: map[string]bool{
				"vcenter1.example.com": true,
			},
		},
		{
			name: "multiple pools on same vcenter deduplicates",
			assignedPools: []*v1.Pool{
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
					},
				},
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
					},
				},
			},
			expected: map[string]bool{
				"vcenter1.example.com": true,
			},
		},
		{
			name: "pools on different vcenters returns all servers",
			assignedPools: []*v1.Pool{
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
					},
				},
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
					},
				},
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter3.example.com",
							},
						},
					},
				},
			},
			expected: map[string]bool{
				"vcenter1.example.com": true,
				"vcenter2.example.com": true,
				"vcenter3.example.com": true,
			},
		},
		{
			name: "pool with empty server is skipped",
			assignedPools: []*v1.Pool{
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "",
							},
						},
					},
				},
				{
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
					},
				},
			},
			expected: map[string]bool{
				"vcenter1.example.com": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVCentersInUse(tt.assignedPools)

			if len(result) != len(tt.expected) {
				t.Errorf("GetVCentersInUse() returned %d vcenters, expected %d: got %v, want %v",
					len(result), len(tt.expected), result, tt.expected)
			}

			for vcenter, expectedVal := range tt.expected {
				if result[vcenter] != expectedVal {
					t.Errorf("GetVCentersInUse() missing expected vcenter %q", vcenter)
				}
			}
		})
	}
}
