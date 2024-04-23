package controller

import (
	"context"
	"errors"
	"fmt"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/resources"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// lastErrorTracker tracks the last error that occurred during reconciliation.
type lastErrorTracker struct {
	// lastError is the last error that occurred during reconciliation.
	lastError error

	// lastErrorTime is the time at which the last error occurred.
	lastErrorTime metav1.Time

	// count is the number of times we've observed the same error in a row.
	count int
}

func (l *LeaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.ResourceRequest{}).
		Complete(l); err != nil {
		return fmt.Errorf("error setting up controller: %w", err)
	}

	// Set up API helpers from the manager.
	l.Client = mgr.GetClient()
	l.Scheme = mgr.GetScheme()
	l.Recorder = mgr.GetEventRecorderFor("control-plane-machine-set-controller")
	l.RESTMapper = mgr.GetRESTMapper()

	return nil
}

func (l *LeaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "namespace", req.Namespace, "name", req.Name)
	logger.V(1).Info("Reconciling resource request")
	defer logger.V(1).Info("Finished reconciling resource request")

	// Fetch the ResourceRequest instance.
	resourceRequest := &v1.ResourceRequest{}
	if err := l.Get(ctx, req.NamespacedName, resourceRequest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TO-DO: clean up the resource request if it is being deleted
	if resourceRequest.DeletionTimestamp != nil {
		logger.V(1).Info("Resource request is being deleted")
		return ctrl.Result{}, nil
	}

	if resourceRequest.Status.Phase == v1.PHASE_FULFILLED ||
		resourceRequest.Status.Phase == v1.PHASE_FAILED {
		return ctrl.Result{}, nil
	}

	leases, err := resources.AcquireLease(resourceRequest)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error acquiring lease: %w", err)
	}

	defer l.Client.Status().Update(ctx, resourceRequest)

	for _, lease := range *leases {
		if err := l.Client.Create(ctx, lease); err != nil {
			message := fmt.Sprintf("error creating lease: %v", err)
			resourceRequest.Status.Phase = v1.PHASE_FAILED
			resourceRequest.Status.State = v1.State(message)
			return ctrl.Result{}, errors.New(message)
		}
	}

	return ctrl.Result{}, nil
}
