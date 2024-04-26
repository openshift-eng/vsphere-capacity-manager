package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceRequestReconciler struct {
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

func (l *ResourceRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.ResourceRequest{}).
		Complete(l); err != nil {
		return fmt.Errorf("error setting up controller: %w", err)
	}

	// Set up API helpers from the manager.
	l.Client = mgr.GetClient()
	l.Scheme = mgr.GetScheme()
	l.Recorder = mgr.GetEventRecorderFor("lease-controller")
	l.RESTMapper = mgr.GetRESTMapper()

	return nil
}

func (l *ResourceRequestReconciler) ensureLeasesAreRemovedFromPool(ctx context.Context, resourceRequest *v1.ResourceRequest) (ctrl.Result, error) {
	for _, leaseRef := range resourceRequest.Status.Leases {
		lease := &v1.Lease{}
		if err := l.Get(ctx, client.ObjectKey{
			Namespace: resourceRequest.Namespace,
			Name:      leaseRef.Name,
		}, lease); err != nil {
			log.Printf("unable to retrieve lease %s: %v", leaseRef.Name, err)
			return ctrl.Result{}, err
		}
		err := l.Client.Delete(ctx, lease)
		if err != nil {
			log.Printf("unable to delete lease, requeuing %s: %v", leaseRef.Name, err)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (l *ResourceRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Fetch the ResourceRequest instance.
	resourceRequest := &v1.ResourceRequest{}
	if err := l.Get(ctx, req.NamespacedName, resourceRequest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//spew.Dump(resourceRequest)
	// TO-DO: clean up the resource request if it is being deleted
	if resourceRequest.DeletionTimestamp != nil {
		return l.ensureLeasesAreRemovedFromPool(ctx, resourceRequest)
	}

	if resourceRequest.Status.Phase == v1.PHASE_FULFILLED ||
		resourceRequest.Status.Phase == v1.PHASE_FAILED {
		return ctrl.Result{}, nil
	}

	defer l.Status().Update(ctx, resourceRequest)

	leases := resources.ConstructLeases(resourceRequest)

	for _, lease := range leases {
		if err := l.Create(ctx, lease); err != nil {
			message := fmt.Sprintf("error creating lease: %v", err)
			resourceRequest.Status.Phase = v1.PHASE_FAILED
			resourceRequest.Status.State = v1.State(message)
			return ctrl.Result{}, errors.New(message)
		}

		resourceRequest.Status.Leases = append(resourceRequest.Status.Leases, corev1.TypedLocalObjectReference{
			Name: lease.Name,
		})
	}

	resourceRequest.Status.Phase = v1.PHASE_FULFILLED

	return ctrl.Result{}, nil
}
