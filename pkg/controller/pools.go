package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PoolReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
	RESTMapper     meta.RESTMapper
	UncachedClient client.Client

	// Namespace is the namespace in which the ControlPlaneMachineSet controller should operate.
	// Any ControlPlaneMachineSet not in this namespace should be ignored.
	Namespace string

	// OperatorName is the name of the ClusterOperator with which the controller should report
	// its status.
	OperatorName string

	// ReleaseVersion is the version of current cluster operator release.
	ReleaseVersion string

	// lastError allows us to track the last error that occurred during reconciliation.
	lastError *lastErrorTracker
}

func (l *PoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pool{}).
		//For(&v1.Lease{}).
		Complete(l); err != nil {
		return fmt.Errorf("error setting up controller: %w", err)
	}

	// Set up API helpers from the manager.
	l.Client = mgr.GetClient()
	l.Scheme = mgr.GetScheme()
	l.Recorder = mgr.GetEventRecorderFor("pools-controller")
	l.RESTMapper = mgr.GetRESTMapper()

	return nil
}

func comparePoolStatus(a, b v1.PoolStatus) bool {
	if a.VCpusAvailable != b.VCpusAvailable ||
		a.MemoryAvailable != b.MemoryAvailable ||
		a.DatastoreAvailable != b.DatastoreAvailable ||
		a.NetworkAvailable != b.NetworkAvailable {
		return false
	}

	if !reflect.DeepEqual(a.Leases, b.Leases) {
		return false
	}

	if !reflect.DeepEqual(a.PortGroups, b.PortGroups) {
		return false
	}

	if !reflect.DeepEqual(a.ActivePortGroups, b.ActivePortGroups) {
		return false
	}

	return true
}

func (l *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespace", req.Namespace, "name", req.Name)
	logger.V(1).Info("Reconciling resource request")
	defer logger.V(1).Info("Finished reconciling resource request")

	// Fetch the ResourceRequest instance.
	pool := &v1.Pool{}
	if err := l.Get(ctx, req.NamespacedName, pool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusOnStart := pool.Status.DeepCopy()

	if pool.DeletionTimestamp != nil {
		logger.V(1).Info("Pool is being deleted")
		return ctrl.Result{}, nil
	}

	resources.AddPool(pool)

	leases := &v1.LeaseList{}
	err := l.Client.List(ctx, leases, client.InNamespace(pool.Namespace))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing leases: %w", err)
	}

	pool.Status.Leases = []corev1.TypedLocalObjectReference{}
	leaseMap := map[string]*v1.Lease{}
	for _, lease := range leases.Items {
		if lease.Status.Pool != nil {
			continue
		}
		pool.Status.Leases = append(pool.Status.Leases, corev1.TypedLocalObjectReference{
			Name: lease.Name,
		})
		leaseMap[lease.Name] = &lease
	}

	resources.CalculateResourceUsage(pool, leaseMap)

	if !comparePoolStatus(*statusOnStart, pool.Status) {
		if err := l.Status().Update(ctx, pool); err != nil {
			return ctrl.Result{RequeueAfter: 2 * time.Second}, fmt.Errorf("error updating pool status: %w", err)
		}
	}
	return ctrl.Result{}, nil
}
