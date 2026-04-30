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
		name     string
		lease    *v1.Lease
		network  *v1.Network
		want     bool
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
