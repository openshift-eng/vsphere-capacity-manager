package controller

import (
	"context"
	"fmt"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"log"
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
}

func (l *PoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pool{}).
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

func (l *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Print("Reconciling pool")
	defer log.Print("Finished reconciling pool")

	// Fetch the Lease instance.
	pool := &v1.Pool{}
	if err := l.Get(ctx, req.NamespacedName, pool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var leases v1.LeaseList
	err := l.Client.List(ctx, &leases, client.InNamespace(req.Namespace))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list leases: %w", err)
	}

	vcpus := 0
	memory := 0
	networks := 0
	for _, lease := range leases.Items {
		if lease.Status.Phase != v1.PHASE_FULFILLED || lease.DeletionTimestamp != nil {
			continue
		}
		log.Printf("checking lease status: %+v", lease)
		for _, ownerRef := range lease.OwnerReferences {
			if ownerRef.Kind == pool.Kind && ownerRef.Name == pool.Name {
				vcpus += lease.Spec.VCpus
				memory += lease.Spec.Memory
				networks += lease.Spec.Networks
				break
			}
		}
	}
	pool.Status.VCpusAvailable = pool.Spec.VCpus - vcpus
	pool.Status.MemoryAvailable = pool.Spec.Memory - memory

	err = l.Client.Status().Update(ctx, pool)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating pool status: %w", err)
	}
	return ctrl.Result{}, nil
}
