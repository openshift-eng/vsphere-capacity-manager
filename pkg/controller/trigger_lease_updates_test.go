package controller

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

// recordingClient is a minimal client.Client stub that only implements Get and
// Update. Any other method call panics via the nil embedded interface, which
// is fine since triggerLeaseUpdates only calls Get and Update.
type recordingClient struct {
	client.Client
	updated  []string
	getCalls []string
}

func (r *recordingClient) Get(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
	r.getCalls = append(r.getCalls, key.Name)

	// If the pre-Get phase filter in triggerLeaseUpdates ever regresses and
	// lets the Failed lease through, simulate a hazardous read returning
	// fresh server-side data for it so the corruption is unmistakable rather
	// than silently leaving obj untouched. The Pending lease's Get call stays
	// a true pass-through, so its cached data is left as Pending.
	if key.Name == "failed-lease" {
		if lease, ok := obj.(*v1.Lease); ok {
			lease.Status = v1.LeaseStatus{Phase: v1.PHASE_FAILED}
			lease.Status.Name = "corrupted-by-get"
		}
	}
	return nil
}

func (r *recordingClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	r.updated = append(r.updated, obj.GetName())
	return nil
}

// TestTriggerLeaseUpdates_SkipsFailedLease guards against a regression where a
// lease that was just transitioned to Failed got picked back up as the
// "oldest" lease needing a forced update, causing it to be re-touched (and, via
// a stale cache read on the shared pointer, potentially re-reconciled) forever
// instead of settling into its terminal state.
func TestTriggerLeaseUpdates_SkipsFailedLease(t *testing.T) {
	now := metav1.Now()
	older := metav1.NewTime(now.Add(-time.Minute))

	failedLease := &v1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failed-lease",
			Namespace:         "default",
			CreationTimestamp: older,
		},
		Spec:   v1.LeaseSpec{NetworkType: v1.NetworkTypeSingleTenant},
		Status: v1.LeaseStatus{Phase: v1.PHASE_FAILED},
	}
	pendingLease := &v1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pending-lease",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec:   v1.LeaseSpec{NetworkType: v1.NetworkTypeSingleTenant},
		Status: v1.LeaseStatus{Phase: v1.PHASE_PENDING},
	}

	restore := setupTestLeases(map[string]*v1.Lease{
		"default/failed-lease":  failedLease,
		"default/pending-lease": pendingLease,
	})
	defer restore()

	stub := &recordingClient{}
	reconciler := &LeaseReconciler{Client: stub}

	reconciler.triggerLeaseUpdates(context.Background(), v1.NetworkTypeSingleTenant)

	for _, name := range stub.updated {
		if name == failedLease.Name {
			t.Errorf("expected Failed lease %s to never be selected for a forced update", failedLease.Name)
		}
	}

	if len(stub.updated) != 1 || stub.updated[0] != pendingLease.Name {
		t.Errorf("expected only the Pending lease %s to be force-updated, got %v", pendingLease.Name, stub.updated)
	}

	for _, name := range stub.getCalls {
		if name == failedLease.Name {
			t.Errorf("expected Get to never be called for the Failed lease %s", failedLease.Name)
		}
	}

	if failedLease.Status.Phase != v1.PHASE_FAILED || failedLease.Status.Name == "corrupted-by-get" {
		t.Errorf("expected the Failed lease to remain unchanged by Get, got status %+v", failedLease.Status)
	}
}
