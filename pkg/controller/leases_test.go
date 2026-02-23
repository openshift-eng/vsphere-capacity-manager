package controller

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
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
