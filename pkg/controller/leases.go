package controller

import (
	"context"
	"fmt"
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

type LeaseReconciler struct {
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

func (l *LeaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.Lease{}).
		Complete(l); err != nil {
		return fmt.Errorf("error setting up controller: %w", err)
	}

	// Set up API helpers from the manager.
	l.Client = mgr.GetClient()
	l.Scheme = mgr.GetScheme()
	l.Recorder = mgr.GetEventRecorderFor("leases-controller")
	l.RESTMapper = mgr.GetRESTMapper()

	return nil
}

func (l *LeaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespace", req.Namespace, "name", req.Name)
	logger.V(1).Info("Reconciling lease")
	defer logger.V(1).Info("Finished reconciling lease")

	// Fetch the Lease instance.
	lease := &v1.Lease{}
	if err := l.Get(ctx, req.NamespacedName, lease); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if lease.DeletionTimestamp != nil {
		logger.V(1).Info("Lease is being deleted")
		err := resources.RemoveLease(lease)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if lease.Status.Pool != nil {
		logger.V(1).Info("Lease already has a pool")
		return ctrl.Result{}, nil
	}

	pool, err := resources.GetPoolWithStrategy(lease, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
	if err != nil {
		return ctrl.Result{}, err
	}

	lease.Status.Pool = &corev1.TypedLocalObjectReference{
		Name: pool.Name,
	}
	err = resources.AddLease(lease)
	if err != nil {
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, err
	}

	err = l.Client.Status().Update(ctx, lease)
	if err != nil {
		return ctrl.Result{}, err
	}

	l.Client.Get(ctx, client.ObjectKeyFromObject(pool), pool)
	return ctrl.Result{}, nil
}
