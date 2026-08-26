package controller

import (
	"strings"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

func testPoolForSatisfiability(name, server string) *v1.Pool {
	return &v1.Pool{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.PoolSpec{
			FailureDomainSpec: v1.FailureDomainSpec{
				VSpherePlatformFailureDomainSpec: configv1.VSpherePlatformFailureDomainSpec{
					Server: server,
				},
			},
		},
	}
}

func TestFailLeaseIfUnsatisfiable(t *testing.T) {
	t.Run("unsatisfiable lease is transitioned to Failed", func(t *testing.T) {
		lease := &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: "impossible-lease"},
			Spec:       v1.LeaseSpec{Pools: 4, VCenters: 1},
			Status: v1.LeaseStatus{
				Phase: v1.PHASE_PARTIAL,
				PoolInfo: []v1.FailureDomainSpec{
					{ShortName: "vc1-pool1"},
				},
			},
		}
		lease.OwnerReferences = []metav1.OwnerReference{
			{Kind: "Pool", Name: "vc1-pool1"},
			{Kind: "Network", Name: "net-1"},
			{Kind: "SomeOtherKind", Name: "keep-me"},
		}

		pools := []*v1.Pool{
			testPoolForSatisfiability("vc1-pool1", "vcenter1.example.com"),
			testPoolForSatisfiability("vc1-pool2", "vcenter1.example.com"),
			testPoolForSatisfiability("vc1-pool3", "vcenter1.example.com"),
		}

		failed := failLeaseIfUnsatisfiable(lease, pools)
		if !failed {
			t.Fatalf("expected failLeaseIfUnsatisfiable to return true")
		}
		if lease.Status.Phase != v1.PHASE_FAILED {
			t.Errorf("expected Phase to be Failed, got %s", lease.Status.Phase)
		}
		if lease.Status.PoolInfo != nil {
			t.Errorf("expected PoolInfo to be cleared, got %v", lease.Status.PoolInfo)
		}
		if len(lease.Status.Topology.Networks) == 0 {
			t.Errorf("expected Topology.Networks to remain non-empty (status.topology.networks is a "+
				"required, minItems:1 field on the CRD; an empty value makes the status update that "+
				"persists Failed get rejected by the API server), got %v", lease.Status.Topology.Networks)
		}
		if len(lease.OwnerReferences) != 1 || lease.OwnerReferences[0].Kind != "SomeOtherKind" {
			t.Errorf("expected Pool/Network owner refs to be stripped, got %v", lease.OwnerReferences)
		}

		var fulfilled *v1.Condition
		for i := range lease.Status.Conditions {
			if lease.Status.Conditions[i].Type == v1.LeaseConditionTypeFulfilled {
				fulfilled = &lease.Status.Conditions[i]
				break
			}
		}
		if fulfilled == nil {
			t.Fatalf("expected a Fulfilled condition to be set")
		}
		if fulfilled.Status != v1.ConditionFalse {
			t.Errorf("expected Fulfilled condition to be False, got %s", fulfilled.Status)
		}
		if fulfilled.Reason != v1.ReasonLeaseUnschedulable {
			t.Errorf("expected reason %s, got %s", v1.ReasonLeaseUnschedulable, fulfilled.Reason)
		}
	})

	t.Run("satisfiable lease is left untouched", func(t *testing.T) {
		lease := &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: "fine-lease"},
			Spec:       v1.LeaseSpec{Pools: 2, VCenters: 1},
			Status:     v1.LeaseStatus{Phase: v1.PHASE_PENDING},
		}

		pools := []*v1.Pool{
			testPoolForSatisfiability("vc1-pool1", "vcenter1.example.com"),
			testPoolForSatisfiability("vc1-pool2", "vcenter1.example.com"),
		}

		failed := failLeaseIfUnsatisfiable(lease, pools)
		if failed {
			t.Fatalf("expected failLeaseIfUnsatisfiable to return false for a satisfiable lease")
		}
		if lease.Status.Phase != v1.PHASE_PENDING {
			t.Errorf("expected Phase to remain unchanged, got %s", lease.Status.Phase)
		}
	})

	t.Run("lease requiring a pool that does not exist is failed", func(t *testing.T) {
		lease := &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: "ghost-pool-lease"},
			Spec:       v1.LeaseSpec{Pools: 1, RequiredPool: "ghost-pool"},
			Status:     v1.LeaseStatus{Phase: v1.PHASE_PENDING},
		}

		pools := []*v1.Pool{
			testPoolForSatisfiability("vc1-pool1", "vcenter1.example.com"),
		}

		if !failLeaseIfUnsatisfiable(lease, pools) {
			t.Fatalf("expected failLeaseIfUnsatisfiable to return true for a nonexistent required pool")
		}
		if lease.Status.Phase != v1.PHASE_FAILED {
			t.Errorf("expected Phase to be Failed, got %s", lease.Status.Phase)
		}

		fulfilled := conditionOfType(lease, v1.LeaseConditionTypeFulfilled)
		if fulfilled == nil {
			t.Fatalf("expected a Fulfilled condition to be set")
		}
		if !strings.Contains(fulfilled.Message, `required pool "ghost-pool" does not exist`) {
			t.Errorf("expected condition message to explain the missing pool, got %q", fulfilled.Message)
		}
	})

	t.Run("lease requiring a disabled pool is failed", func(t *testing.T) {
		disabledPool := testPoolForSatisfiability("disabled-pool", "vcenter1.example.com")
		disabledPool.Spec.NoSchedule = true

		lease := &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: "disabled-pool-lease"},
			Spec:       v1.LeaseSpec{Pools: 1, RequiredPool: "disabled-pool"},
			Status:     v1.LeaseStatus{Phase: v1.PHASE_PENDING},
		}

		pools := []*v1.Pool{disabledPool}

		if !failLeaseIfUnsatisfiable(lease, pools) {
			t.Fatalf("expected failLeaseIfUnsatisfiable to return true for a disabled required pool")
		}
		if lease.Status.Phase != v1.PHASE_FAILED {
			t.Errorf("expected Phase to be Failed, got %s", lease.Status.Phase)
		}

		fulfilled := conditionOfType(lease, v1.LeaseConditionTypeFulfilled)
		if fulfilled == nil {
			t.Fatalf("expected a Fulfilled condition to be set")
		}
		if !strings.Contains(fulfilled.Message, `required pool "disabled-pool" is disabled`) {
			t.Errorf("expected condition message to explain the disabled pool, got %q", fulfilled.Message)
		}
	})
}

// conditionOfType returns the lease condition of the given type, or nil if not present.
func conditionOfType(lease *v1.Lease, condType v1.ConditionType) *v1.Condition {
	for i := range lease.Status.Conditions {
		if lease.Status.Conditions[i].Type == condType {
			return &lease.Status.Conditions[i]
		}
	}
	return nil
}

func TestShouldLeaseBeDelayed_SkipsFailed(t *testing.T) {
	now := metav1.Now()
	older := metav1.NewTime(now.Add(-time.Minute))

	newLease := &v1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "new-lease", CreationTimestamp: now},
		Spec:       v1.LeaseSpec{NetworkType: v1.NetworkTypeSingleTenant},
		Status:     v1.LeaseStatus{Phase: v1.PHASE_PENDING},
	}

	t.Run("an older Failed lease does not delay a newer Pending lease", func(t *testing.T) {
		failedLease := &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: "failed-lease", CreationTimestamp: older},
			Spec:       v1.LeaseSpec{NetworkType: v1.NetworkTypeSingleTenant},
			Status:     v1.LeaseStatus{Phase: v1.PHASE_FAILED},
		}

		restore := setupTestLeases(map[string]*v1.Lease{
			"default/new-lease":    newLease,
			"default/failed-lease": failedLease,
		})
		defer restore()

		if shouldLeaseBeDelayed(newLease) {
			t.Errorf("expected a Failed lease to not block a newer Pending lease")
		}
	})

	t.Run("an older Partial lease still delays a newer Pending lease", func(t *testing.T) {
		partialLease := &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: "partial-lease", CreationTimestamp: older},
			Spec:       v1.LeaseSpec{NetworkType: v1.NetworkTypeSingleTenant},
			Status:     v1.LeaseStatus{Phase: v1.PHASE_PARTIAL},
		}

		restore := setupTestLeases(map[string]*v1.Lease{
			"default/new-lease":     newLease,
			"default/partial-lease": partialLease,
		})
		defer restore()

		if !shouldLeaseBeDelayed(newLease) {
			t.Errorf("expected a Partial lease to still block a newer Pending lease")
		}
	})
}
