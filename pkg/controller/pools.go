package controller

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/resources"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (l *PoolReconciler) ensureCapacityIsUpdated(ctx context.Context, pool *v1.Pool) error {
	leases := &v1.LeaseList{}
	pool.Status.NetworkAvailable = len(pool.Status.PortGroups)
	pool.Status.VCpusAvailable = pool.Spec.VCpus
	pool.Status.MemoryAvailable = pool.Spec.Memory
	pool.Status.DatastoreAvailable = pool.Spec.Storage

	err := l.Client.List(ctx, leases, client.InNamespace(pool.Namespace))
	if err != nil {
		return fmt.Errorf("error listing leases: %w", err)
	}

	for _, lease := range leases.Items {
		if lease.Status.Pool == nil {
			continue
		}
		if lease.Status.Pool.Name != pool.Name {
			continue
		}
		resources.CalculateResourceUsage(pool, &lease)
	}

	return l.Client.Status().Update(ctx, pool)
}

func (l *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//logger := log.FromContext(ctx, "namespace", req.Namespace, "name", req.Name)
	log.Printf("reconciling pool: %s", req.Name)
	//logger.V(1).Info("Reconciling resource request")
	defer log.Print("Finished reconciling resource request")

	// Fetch the ResourceRequest instance.
	pool := &v1.Pool{}
	if err := l.Get(ctx, req.NamespacedName, pool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Printf("pool %s annotations: %v", pool.Name, pool.Annotations)

	if pool.DeletionTimestamp != nil {
		log.Print("Pool is being deleted")
		return ctrl.Result{}, nil
	}

	leases := &v1.LeaseList{}
	err := l.Client.List(ctx, leases, client.InNamespace(pool.Namespace))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing leases: %w", err)
	}

	err = l.ensureCapacityIsUpdated(ctx, pool)
	if err != nil {
		log.Printf("error updating pool status, requeuing: %v", err)
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}
