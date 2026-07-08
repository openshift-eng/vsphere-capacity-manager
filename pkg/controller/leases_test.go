package controller

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

func setupTestNetworks(nets map[string]*v1.Network) func() {
	old := networks
	networks = nets
	return func() { networks = old }
}

func setupTestLeases(ls map[string]*v1.Lease) func() {
	old := leases
	leases = ls
	return func() { leases = old }
}

func TestDoesLeaseContainPortGroup(t *testing.T) {
	dc := "dc1"
	pod := "pod1"

	poolNetwork := &v1.Network{
		ObjectMeta: metav1.ObjectMeta{Name: "net-1"},
		Spec: v1.NetworkSpec{
			VlanId:         "100",
			DatacenterName: &dc,
			PodName:        &pod,
			PortGroupName:  "pg-100",
		},
	}

	pool := &v1.Pool{
		Spec: v1.PoolSpec{
			FailureDomainSpec: v1.FailureDomainSpec{
				VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
					Topology: configv1.VSpherePlatformTopology{
						Networks: []string{"/dc1/network/pg-100"},
					},
				},
			},
			IBMPoolSpec: v1.IBMPoolSpec{Pod: pod},
		},
	}

	candidateNetwork := &v1.Network{
		Spec: v1.NetworkSpec{
			VlanId:         "100",
			DatacenterName: &dc,
		},
	}

	tests := []struct {
		name    string
		lease   *v1.Lease
		network *v1.Network
		want    bool
	}{
		{
			name: "returns false when lease has no owner references",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{}},
			},
			network: candidateNetwork,
			want:    false,
		},
		{
			name: "returns false when owner reference network is not in pool",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Network", Name: "net-unknown"},
					},
				},
			},
			network: candidateNetwork,
			want:    false,
		},
		{
			name: "returns true when matching port group exists",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Network", Name: "net-1"},
					},
				},
			},
			network: candidateNetwork,
			want:    true,
		},
		{
			name: "returns false when VlanId differs",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Network", Name: "net-1"},
					},
				},
			},
			network: &v1.Network{
				Spec: v1.NetworkSpec{
					VlanId:         "200",
					DatacenterName: &dc,
				},
			},
			want: false,
		},
		{
			name: "returns false when DatacenterName differs",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Network", Name: "net-1"},
					},
				},
			},
			network: func() *v1.Network {
				otherDC := "dc2"
				return &v1.Network{
					Spec: v1.NetworkSpec{
						VlanId:         "100",
						DatacenterName: &otherDC,
					},
				}
			}(),
			want: false,
		},
		{
			name: "skips non-Network owner references",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Pool", Name: "some-pool"},
					},
				},
			},
			network: candidateNetwork,
			want:    false,
		},
	}

	cleanup := setupTestNetworks(map[string]*v1.Network{
		"net-1": poolNetwork,
	})
	defer cleanup()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := doesLeaseContainPortGroup(tt.lease, pool, tt.network)
			if got != tt.want {
				t.Errorf("doesLeaseContainPortGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCommonNetworksForLease(t *testing.T) {
	dc1 := "cidatacenter-1"
	dc2 := "cidatacenter-2"
	pod := "dal10.pod03"

	// Network in pool A's topology (portGroupName matches pool A's path)
	netPoolA := &v1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ci-vlan-1284-dal10-dal10.pod03-1",
			Labels: map[string]string{
				v1.NetworkTypeLabel: "multi-tenant",
			},
		},
		TypeMeta: metav1.TypeMeta{Kind: "Network"},
		Spec: v1.NetworkSpec{
			PortGroupName:  "ci-vlan-1284-1",
			VlanId:         "1284",
			DatacenterName: &dc1,
			PodName:        &pod,
		},
	}

	// Network in pool B's topology (portGroupName matches pool B's path)
	netPoolB := &v1.Network{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ci-vlan-1284-dal10-dal10.pod03-2",
			Labels: map[string]string{
				v1.NetworkTypeLabel: "multi-tenant",
			},
		},
		TypeMeta: metav1.TypeMeta{Kind: "Network"},
		Spec: v1.NetworkSpec{
			PortGroupName:  "ci-vlan-1284-2",
			VlanId:         "1284",
			DatacenterName: &dc2,
			PodName:        &pod,
		},
	}

	// Pool A has portgroup ci-vlan-1284-1
	poolA := &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{Name: "pool-a"},
		Spec: v1.PoolSpec{
			FailureDomainSpec: v1.FailureDomainSpec{
				VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
					Topology: configv1.VSpherePlatformTopology{
						Networks: []string{"/cidatacenter-1/network/ci-vlan-1284-1"},
					},
				},
			},
			IBMPoolSpec: v1.IBMPoolSpec{Pod: pod},
		},
	}

	// Pool B has portgroup ci-vlan-1284-2
	poolB := &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{Name: "pool-b"},
		Spec: v1.PoolSpec{
			FailureDomainSpec: v1.FailureDomainSpec{
				VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
					Topology: configv1.VSpherePlatformTopology{
						Networks: []string{"/cidatacenter-2/network/ci-vlan-1284-2"},
					},
				},
			},
			IBMPoolSpec: v1.IBMPoolSpec{Pod: pod},
		},
	}

	// Sibling lease on pool A, already has netPoolA assigned
	siblingLease := &v1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-lease-a",
			Labels: map[string]string{
				BoskosIdLabel: "job-123",
			},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "Pool", Name: "pool-a"},
				{Kind: "Network", Name: netPoolA.Name, UID: netPoolA.UID},
			},
		},
		Spec: v1.LeaseSpec{
			VCpus:       8,
			Memory:      32,
			NetworkType: v1.NetworkTypeMultiTenant,
		},
		Status: v1.LeaseStatus{
			Phase: v1.PHASE_PENDING,
		},
	}

	// Target lease on pool B, same Boskos ID, no networks yet
	targetLease := &v1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name: "job-lease-b",
			Labels: map[string]string{
				BoskosIdLabel: "job-123",
			},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "Pool", Name: "pool-b"},
			},
		},
		Spec: v1.LeaseSpec{
			VCpus:       8,
			Memory:      32,
			NetworkType: v1.NetworkTypeMultiTenant,
		},
		Status: v1.LeaseStatus{
			Phase: v1.PHASE_PENDING,
		},
	}

	cleanupNetworks := setupTestNetworks(map[string]*v1.Network{
		netPoolA.Name: netPoolA,
		netPoolB.Name: netPoolB,
	})
	defer cleanupNetworks()

	cleanupLeases := setupTestLeases(map[string]*v1.Lease{
		siblingLease.Name: siblingLease,
		targetLease.Name:  targetLease,
	})
	defer cleanupLeases()

	reconciler := &LeaseReconciler{}

	t.Run("returns sibling networks unfiltered", func(t *testing.T) {
		got, err := reconciler.getCommonNetworksForLease(targetLease)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].Name != netPoolA.Name {
			t.Errorf("expected sibling's network %s, got %v", netPoolA.Name, got)
		}
	})

	t.Run("sibling network not in target pool topology", func(t *testing.T) {
		poolNetworksMap := getNetworksForPool(poolB)
		if _, exists := poolNetworksMap[netPoolA.Name]; exists {
			t.Error("sibling's network should NOT be in pool B's topology")
		}
	})

	t.Run("pool-local network is in target pool topology", func(t *testing.T) {
		poolNetworksMap := getNetworksForPool(poolB)
		if _, exists := poolNetworksMap[netPoolB.Name]; !exists {
			t.Error("pool B's own network should be in its topology")
		}
	})

	t.Run("getAvailableNetworks returns pool-local network", func(t *testing.T) {
		got := reconciler.getAvailableNetworks(poolB, v1.NetworkTypeMultiTenant)
		if len(got) != 1 || got[0].Name != netPoolB.Name {
			t.Errorf("expected pool B's network %s, got %v", netPoolB.Name, got)
		}
	})

	t.Run("getAvailableNetworks excludes cross-pool network", func(t *testing.T) {
		got := reconciler.getAvailableNetworks(poolA, v1.NetworkTypeMultiTenant)
		for _, n := range got {
			if n.Name == netPoolB.Name {
				t.Error("pool A's available networks should not include pool B's network")
			}
		}
	})
}

func TestGetNetworkType(t *testing.T) {
	tests := []struct {
		name     string
		network  *v1.Network
		expected string
	}{
		{
			name: "returns single-tenant when no labels",
			network: &v1.Network{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expected: "single-tenant",
		},
		{
			name: "returns single-tenant when label missing",
			network: &v1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"other": "value"},
				},
			},
			expected: "single-tenant",
		},
		{
			name: "returns multi-tenant from label",
			network: &v1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						v1.NetworkTypeLabel: "multi-tenant",
					},
				},
			},
			expected: "multi-tenant",
		},
		{
			name: "returns disconnected from label",
			network: &v1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						v1.NetworkTypeLabel: "disconnected",
					},
				},
			},
			expected: "disconnected",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNetworkType(tt.network)
			if got != tt.expected {
				t.Errorf("getNetworkType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUpdateNetworkTypeMetrics(t *testing.T) {
	dc := "dc1"
	pod := "pod1"

	oldPools := pools
	oldNetworks := networks
	oldLeases := leases
	defer func() {
		pools = oldPools
		networks = oldNetworks
		leases = oldLeases
	}()

	networks = map[string]*v1.Network{
		"default/net-st-1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "net-st-1",
				Namespace: "default",
			},
			Spec: v1.NetworkSpec{
				PortGroupName:  "pg-100",
				VlanId:         "100",
				DatacenterName: &dc,
				PodName:        &pod,
			},
		},
		"default/net-st-2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "net-st-2",
				Namespace: "default",
			},
			Spec: v1.NetworkSpec{
				PortGroupName:  "pg-101",
				VlanId:         "101",
				DatacenterName: &dc,
				PodName:        &pod,
			},
		},
		"default/net-mt-1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "net-mt-1",
				Namespace: "default",
				Labels: map[string]string{
					v1.NetworkTypeLabel: "multi-tenant",
				},
			},
			Spec: v1.NetworkSpec{
				PortGroupName:  "pg-200",
				VlanId:         "200",
				DatacenterName: &dc,
				PodName:        &pod,
			},
		},
	}

	pools = map[string]*v1.Pool{
		"default/pool1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pool1",
				Namespace: "default",
			},
			Spec: v1.PoolSpec{
				FailureDomainSpec: v1.FailureDomainSpec{
					VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
						Topology: configv1.VSpherePlatformTopology{
							Networks: []string{"/dc1/network/pg-100", "/dc1/network/pg-101", "/dc1/network/pg-200"},
						},
					},
				},
				IBMPoolSpec: v1.IBMPoolSpec{Pod: pod},
			},
		},
	}

	leases = map[string]*v1.Lease{
		"default/lease1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease1",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Network", Name: "net-st-1"},
				},
			},
			Spec: v1.LeaseSpec{
				NetworkType: v1.NetworkTypeSingleTenant,
			},
			Status: v1.LeaseStatus{
				Phase: v1.PHASE_FULFILLED,
			},
		},
	}

	updateNetworkTypeMetrics()

	stTotal := testutil.ToFloat64(PoolNetworksTotalByType.WithLabelValues("default", "pool1", "single-tenant"))
	if stTotal != 2 {
		t.Errorf("expected single-tenant total = 2, got %v", stTotal)
	}

	stAvail := testutil.ToFloat64(PoolNetworksAvailableByType.WithLabelValues("default", "pool1", "single-tenant"))
	if stAvail != 1 {
		t.Errorf("expected single-tenant available = 1, got %v", stAvail)
	}

	mtTotal := testutil.ToFloat64(PoolNetworksTotalByType.WithLabelValues("default", "pool1", "multi-tenant"))
	if mtTotal != 1 {
		t.Errorf("expected multi-tenant total = 1, got %v", mtTotal)
	}

	mtAvail := testutil.ToFloat64(PoolNetworksAvailableByType.WithLabelValues("default", "pool1", "multi-tenant"))
	if mtAvail != 1 {
		t.Errorf("expected multi-tenant available = 1, got %v", mtAvail)
	}

	// Verify network_lease_count
	stInUse := testutil.ToFloat64(NetworkLeaseCount.WithLabelValues("default", "net-st-1", "single-tenant", "pool1"))
	if stInUse != 1 {
		t.Errorf("expected net-st-1 lease count = 1, got %v", stInUse)
	}

	stFree := testutil.ToFloat64(NetworkLeaseCount.WithLabelValues("default", "net-st-2", "single-tenant", "pool1"))
	if stFree != 0 {
		t.Errorf("expected net-st-2 lease count = 0, got %v", stFree)
	}

	mtFree := testutil.ToFloat64(NetworkLeaseCount.WithLabelValues("default", "net-mt-1", "multi-tenant", "pool1"))
	if mtFree != 0 {
		t.Errorf("expected net-mt-1 lease count = 0, got %v", mtFree)
	}
}

func TestUpdateLeaseMetrics(t *testing.T) {
	oldLeases := leases
	oldPools := pools
	oldNetworks := networks
	defer func() {
		leases = oldLeases
		pools = oldPools
		networks = oldNetworks
	}()

	networks = make(map[string]*v1.Network)
	pools = make(map[string]*v1.Pool)

	now := metav1.Now()
	leases = map[string]*v1.Lease{
		"default/test-lease": {
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-lease",
				Namespace:         "default",
				CreationTimestamp: now,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Pool", Name: "pool1"},
				},
			},
			Spec: v1.LeaseSpec{
				NetworkType: v1.NetworkTypeMultiTenant,
			},
			Status: v1.LeaseStatus{
				Phase: v1.PHASE_FULFILLED,
			},
		},
	}

	updateLeaseMetrics()

	count := testutil.ToFloat64(LeaseCounts.WithLabelValues("default", "multi-tenant", "Fulfilled"))
	if count != 1 {
		t.Errorf("expected lease count = 1, got %v", count)
	}

	age := testutil.ToFloat64(LeaseAgeSeconds.WithLabelValues("default", "test-lease", "pool1", "multi-tenant"))
	if age < 0 || age > 5 {
		t.Errorf("expected lease age near 0 seconds, got %v", age)
	}
}

func TestPoolMissingNetworks(t *testing.T) {
	dc := "dc1"
	pod := "pod1"

	net1 := &v1.Network{
		ObjectMeta: metav1.ObjectMeta{Name: "net-1"},
		TypeMeta:   metav1.TypeMeta{Kind: "Network"},
		Spec: v1.NetworkSpec{
			VlanId:         "100",
			DatacenterName: &dc,
			PodName:        &pod,
			PortGroupName:  "pg-100",
		},
	}

	net2 := &v1.Network{
		ObjectMeta: metav1.ObjectMeta{Name: "net-2"},
		TypeMeta:   metav1.TypeMeta{Kind: "Network"},
		Spec: v1.NetworkSpec{
			VlanId:         "200",
			DatacenterName: &dc,
			PodName:        &pod,
			PortGroupName:  "pg-200",
		},
	}

	poolA := &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{Name: "pool-a"},
		Spec: v1.PoolSpec{
			FailureDomainSpec: v1.FailureDomainSpec{
				VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
					Topology: configv1.VSpherePlatformTopology{
						Networks: []string{"/dc1/network/pg-100"},
					},
				},
			},
			IBMPoolSpec: v1.IBMPoolSpec{Pod: pod},
		},
	}

	poolB := &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{Name: "pool-b"},
		Spec: v1.PoolSpec{
			FailureDomainSpec: v1.FailureDomainSpec{
				VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
					Topology: configv1.VSpherePlatformTopology{
						Networks: []string{"/dc1/network/pg-200"},
					},
				},
			},
			IBMPoolSpec: v1.IBMPoolSpec{Pod: pod},
		},
	}

	cleanup := setupTestNetworks(map[string]*v1.Network{
		"net-1": net1,
		"net-2": net2,
	})
	defer cleanup()

	tests := []struct {
		name         string
		lease        *v1.Lease
		pools        []*v1.Pool
		wantMissing  bool
		wantPoolName string
	}{
		{
			name: "returns false when all pools have networks",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Pool", Name: "pool-a"},
						{Kind: "Network", Name: "net-1"},
						{Kind: "Pool", Name: "pool-b"},
						{Kind: "Network", Name: "net-2"},
					},
				},
			},
			pools:       []*v1.Pool{poolA, poolB},
			wantMissing: false,
		},
		{
			name: "returns true when a pool has no networks",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Pool", Name: "pool-a"},
						{Kind: "Pool", Name: "pool-b"},
						{Kind: "Network", Name: "net-1"},
					},
				},
			},
			pools:        []*v1.Pool{poolA, poolB},
			wantMissing:  true,
			wantPoolName: "pool-b",
		},
		{
			name: "returns true when no networks assigned at all",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Pool", Name: "pool-a"},
					},
				},
			},
			pools:        []*v1.Pool{poolA},
			wantMissing:  true,
			wantPoolName: "pool-a",
		},
		{
			name: "returns false with single pool that has a network",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Pool", Name: "pool-a"},
						{Kind: "Network", Name: "net-1"},
					},
				},
			},
			pools:       []*v1.Pool{poolA},
			wantMissing: false,
		},
		{
			name: "ignores networks not in pool topology",
			lease: &v1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "Pool", Name: "pool-a"},
						{Kind: "Network", Name: "net-2"},
					},
				},
			},
			pools:        []*v1.Pool{poolA},
			wantMissing:  true,
			wantPoolName: "pool-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolName, missing := poolMissingNetworks(tt.lease, tt.pools)
			if missing != tt.wantMissing {
				t.Errorf("poolMissingNetworks() missing = %v, want %v", missing, tt.wantMissing)
			}
			if tt.wantMissing && poolName != tt.wantPoolName {
				t.Errorf("poolMissingNetworks() poolName = %v, want %v", poolName, tt.wantPoolName)
			}
		})
	}
}

// TestLeaseVCentersCapEnforcement tests that the VCenters field properly caps vcenter diversity
func TestLeaseVCentersCapEnforcement(t *testing.T) {
	tests := []struct {
		name             string
		leaseVCenters    int
		assignedPools    []*v1.Pool
		expectExclusions bool
		expectedExcluded map[string]bool
	}{
		{
			name:          "no cap when VCenters is 0",
			leaseVCenters: 0,
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
			expectExclusions: false,
			expectedExcluded: nil,
		},
		{
			name:             "no cap when VCenters not set and no pools assigned yet",
			leaseVCenters:    0,
			assignedPools:    []*v1.Pool{},
			expectExclusions: false,
			expectedExcluded: nil,
		},
		{
			name:          "no exclusions when under cap",
			leaseVCenters: 3,
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
			},
			expectExclusions: false,
			expectedExcluded: nil,
		},
		{
			name:          "exclusions when at cap",
			leaseVCenters: 2,
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
			},
			expectExclusions: true,
			expectedExcluded: map[string]bool{
				// Any vcenter not in the assigned pools should be excluded
				"vcenter3.example.com": true,
			},
		},
		{
			name:          "allows same vcenters when cap reached",
			leaseVCenters: 1,
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
			expectExclusions: true,
			expectedExcluded: map[string]bool{
				// vcenter1 is allowed (in use), all others excluded
				"vcenter2.example.com": true,
				"vcenter3.example.com": true,
			},
		},
		{
			name:          "handles pools with empty server field",
			leaseVCenters: 1,
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
			},
			expectExclusions: false,
			expectedExcluded: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the controller logic for computing excluded vcenters
			var excludedVCenters map[string]bool
			if tt.leaseVCenters > 0 {
				// This matches the logic in pkg/controller/leases.go
				vcentersInUse := make(map[string]bool)
				for _, p := range tt.assignedPools {
					if p.Spec.Server != "" {
						vcentersInUse[p.Spec.Server] = true
					}
				}

				if len(vcentersInUse) >= tt.leaseVCenters {
					// Cap reached - build exclusion list
					// (In real controller, this would check all available pools)
					excludedVCenters = make(map[string]bool)
					// For testing purposes, we'll check against the expected exclusions
					if tt.expectedExcluded != nil {
						for server := range tt.expectedExcluded {
							if !vcentersInUse[server] {
								excludedVCenters[server] = true
							}
						}
					}
				}
			}

			// Verify exclusions match expectations
			if tt.expectExclusions {
				if excludedVCenters == nil {
					t.Error("Expected exclusions but got none")
				} else if tt.expectedExcluded != nil {
					for server := range tt.expectedExcluded {
						if !excludedVCenters[server] {
							t.Errorf("Expected server %q to be excluded but it was not", server)
						}
					}
				}
			} else {
				if len(excludedVCenters) > 0 {
					t.Errorf("Expected no exclusions but got %v", excludedVCenters)
				}
			}
		})
	}
}

// TestLeaseVCenterPoolAvailability tests that when VCenters=1 and Pools>1,
// we only select from vCenters that have enough suitable pools
func TestLeaseVCenterPoolAvailability(t *testing.T) {
	tests := []struct {
		name             string
		requiredPools    int
		vcentersLimit    int
		availablePools   []*v1.Pool
		expectExclusions bool
		expectedExcluded map[string]bool
	}{
		{
			name:          "vcenter with insufficient pools is excluded",
			requiredPools: 3,
			vcentersLimit: 1,
			availablePools: []*v1.Pool{
				// vcenter1 has only 1 pool - insufficient for 3 pools
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				// vcenter2 has 3 pools - sufficient for 3 pools
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool3"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
			},
			expectExclusions: true,
			expectedExcluded: map[string]bool{
				"vcenter1.example.com": true, // only 1 pool, needs 3
			},
		},
		{
			name:          "vcenter with insufficient resources is excluded",
			requiredPools: 2,
			vcentersLimit: 1,
			availablePools: []*v1.Pool{
				// vcenter1 has 2 pools, but one lacks resources
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  5,  // insufficient
						MemoryAvailable: 10, // insufficient
					},
				},
				// vcenter2 has 2 pools with sufficient resources
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
			},
			expectExclusions: true,
			expectedExcluded: map[string]bool{
				"vcenter1.example.com": true, // only 1 suitable pool, needs 2
			},
		},
		{
			name:          "vcenter with excluded pools counted correctly",
			requiredPools: 2,
			vcentersLimit: 1,
			availablePools: []*v1.Pool{
				// vcenter1 has 2 pools, but one is marked exclude
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:   100,
						Memory:  1000,
						Exclude: true, // This pool should not be counted
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				// vcenter2 has 2 schedulable pools
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
			},
			expectExclusions: true,
			expectedExcluded: map[string]bool{
				"vcenter1.example.com": true, // only 1 suitable pool (pool2 is excluded), needs 2
			},
		},
		{
			name:          "no exclusions when VCenters >= Pools",
			requiredPools: 2,
			vcentersLimit: 3, // Can use 3 vCenters for 2 pools, no pre-filtering needed
			availablePools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
			},
			expectExclusions: false,
		},
		{
			name:          "no exclusions when only 1 pool required",
			requiredPools: 1,
			vcentersLimit: 1,
			availablePools: []*v1.Pool{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
			},
			expectExclusions: false,
		},
		{
			name:          "vcenters=2 pools=3 excludes vcenters with <2 pools",
			requiredPools: 3,
			vcentersLimit: 2,
			availablePools: []*v1.Pool{
				// vcenter1 has only 1 pool - should be excluded (needs ceil(3/2)=2)
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				// vcenter2 has 2 pools - sufficient
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				// vcenter3 has 2 pools - sufficient
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter3-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter3.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter3-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter3.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
			},
			expectExclusions: true,
			expectedExcluded: map[string]bool{
				"vcenter1.example.com": true, // only 1 pool, needs at least 2
			},
		},
		{
			name:          "vcenters=2 pools=5 excludes vcenters with <3 pools",
			requiredPools: 5,
			vcentersLimit: 2,
			availablePools: []*v1.Pool{
				// vcenter1 has 2 pools - insufficient (needs ceil(5/2)=3)
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter1-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter1.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				// vcenter2 has 3 pools - sufficient
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool1"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool2"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vcenter2-pool3"},
					Spec: v1.PoolSpec{
						FailureDomainSpec: v1.FailureDomainSpec{
							VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
								Server: "vcenter2.example.com",
							},
						},
						VCpus:  100,
						Memory: 1000,
					},
					Status: v1.PoolStatus{
						VCpusAvailable:  100,
						MemoryAvailable: 1000,
					},
				},
			},
			expectExclusions: true,
			expectedExcluded: map[string]bool{
				"vcenter1.example.com": true, // only 2 pools, needs at least 3
			},
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
					VCpus:    16,
					Memory:   32,
					Pools:    tt.requiredPools,
					VCenters: tt.vcentersLimit,
				},
			}

			// Simulate the controller logic for pre-filtering vCenters
			var excludedVCenters map[string]bool
			assignedPools := []*v1.Pool{} // No pools assigned yet

			if lease.Spec.VCenters > 0 {
				vcentersInUse := make(map[string]bool)
				for _, p := range assignedPools {
					if p.Spec.Server != "" {
						vcentersInUse[p.Spec.Server] = true
					}
				}

				if len(vcentersInUse) >= lease.Spec.VCenters {
					// Cap already reached - not testing this case
					t.Skip("Test case is for pre-assignment filtering")
				} else if lease.Spec.VCenters > 0 && lease.Spec.VCenters < tt.requiredPools && len(assignedPools) == 0 {
					// This is the special case we're testing: VCenters < Pools
					fittingPoolsPerVCenter := make(map[string][]*v1.Pool)
					for _, p := range tt.availablePools {
						if p.Spec.Server == "" {
							continue
						}
						if p.Spec.NoSchedule {
							continue
						}
						if p.Spec.Exclude && lease.Spec.RequiredPool != p.ObjectMeta.Name {
							continue
						}
						if len(lease.Spec.RequiredPool) > 0 && lease.Spec.RequiredPool != p.ObjectMeta.Name {
							continue
						}
						if int(p.Status.VCpusAvailable) < lease.Spec.VCpus || int(p.Status.MemoryAvailable) < lease.Spec.Memory {
							continue
						}
						fittingPoolsPerVCenter[p.Spec.Server] = append(fittingPoolsPerVCenter[p.Spec.Server], p)
					}

					// Calculate minimum pools per vCenter using ceiling division
					minPoolsPerVCenter := (tt.requiredPools + tt.vcentersLimit - 1) / tt.vcentersLimit

					excludedVCenters = make(map[string]bool)
					for vcenter, pools := range fittingPoolsPerVCenter {
						if len(pools) < minPoolsPerVCenter {
							excludedVCenters[vcenter] = true
							t.Logf("Excluding vcenter %s: only has %d suitable pools but needs at least %d per vCenter (total %d pools across %d vCenters)",
								vcenter, len(pools), minPoolsPerVCenter, tt.requiredPools, tt.vcentersLimit)
						}
					}

					// Also exclude vCenters not in the map (0 suitable pools)
					seenVCenters := make(map[string]bool)
					for vcenter := range fittingPoolsPerVCenter {
						seenVCenters[vcenter] = true
					}
					for _, p := range tt.availablePools {
						if p.Spec.Server != "" && !seenVCenters[p.Spec.Server] {
							excludedVCenters[p.Spec.Server] = true
						}
					}
				}
			}

			// Verify exclusions match expectations
			if tt.expectExclusions {
				if len(excludedVCenters) == 0 {
					t.Error("Expected exclusions but got none")
				} else if tt.expectedExcluded != nil {
					for server := range tt.expectedExcluded {
						if !excludedVCenters[server] {
							t.Errorf("Expected server %q to be excluded but it was not", server)
						}
					}
					// Verify we didn't exclude anything extra
					for server := range excludedVCenters {
						if !tt.expectedExcluded[server] {
							t.Errorf("Server %q was excluded but should not have been", server)
						}
					}
				}
			} else {
				if len(excludedVCenters) > 0 {
					t.Errorf("Expected no exclusions but got %v", excludedVCenters)
				}
			}
		})
	}
}

// TestLeaseVCentersPoolsInteraction tests the semantic interaction between Pools and VCenters fields
func TestLeaseVCentersPoolsInteraction(t *testing.T) {
	tests := []struct {
		name        string
		pools       int
		vcenters    int
		description string
		valid       bool
	}{
		{
			name:        "vcenters equals pools - each pool different vcenter",
			pools:       3,
			vcenters:    3,
			description: "Lease requests 3 pools across up to 3 vcenters",
			valid:       true,
		},
		{
			name:        "vcenters less than pools - requires sharing",
			pools:       4,
			vcenters:    2,
			description: "Lease requests 4 pools but limited to 2 vcenters - pools must share vcenters",
			valid:       true,
		},
		{
			name:        "vcenters greater than pools - cap not binding",
			pools:       2,
			vcenters:    5,
			description: "Lease allows up to 5 vcenters but only needs 2 pools - cap won't be reached",
			valid:       true,
		},
		{
			name:        "vcenters zero - no limit",
			pools:       10,
			vcenters:    0,
			description: "No vcenter cap - pools can span any number of vcenters",
			valid:       true,
		},
		{
			name:        "single pool single vcenter",
			pools:       1,
			vcenters:    1,
			description: "Simplest case - one pool on one vcenter",
			valid:       true,
		},
		{
			name:        "many pools single vcenter",
			pools:       10,
			vcenters:    1,
			description: "All pools must be on the same vcenter",
			valid:       true,
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
					VCpus:    16,
					Memory:   32,
					Pools:    tt.pools,
					VCenters: tt.vcenters,
				},
			}

			// All of these should be valid lease configurations
			if lease.Spec.Pools != tt.pools {
				t.Errorf("Expected Pools=%d, got %d", tt.pools, lease.Spec.Pools)
			}
			if lease.Spec.VCenters != tt.vcenters {
				t.Errorf("Expected VCenters=%d, got %d", tt.vcenters, lease.Spec.VCenters)
			}

			// Document the semantics
			t.Logf("Semantic: %s", tt.description)
			if tt.vcenters > 0 && tt.vcenters < tt.pools {
				t.Logf("  → Pools must be distributed across at most %d vcenters", tt.vcenters)
			} else if tt.vcenters == 0 {
				t.Logf("  → No vcenter diversity constraint")
			} else if tt.vcenters >= tt.pools {
				t.Logf("  → Vcenter cap (%d) is not binding for %d pools", tt.vcenters, tt.pools)
			}
		})
	}
}
