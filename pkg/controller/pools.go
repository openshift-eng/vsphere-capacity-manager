package controller

import (
	"context"
	"fmt"
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
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

	poolKey := fmt.Sprintf("%s/%s", req.Namespace, req.Name)

	// Fetch the Pool instance.
	pool := &v1.Pool{}
	if err := l.Get(ctx, req.NamespacedName, pool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if pool.DeletionTimestamp != nil {
		log.Print("Pool is being deleted")
		if pool.Finalizers != nil {
			pool.Finalizers = nil
			err := l.Update(ctx, pool)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error updating pool: %w", err)
			}
		}
		poolsMu.Lock()
		delete(pools, poolKey)
		poolsMu.Unlock()
		return ctrl.Result{}, nil
	}

	if pool.Finalizers == nil {
		log.Print("setting finalizer on pool")
		pool.Finalizers = []string{v1.PoolFinalizer}
		err := l.Client.Update(ctx, pool)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error setting pool finalizer: %w", err)
		}
	}

	if !pool.Status.Initialized {
		pool.Status.VCpusAvailable = pool.Spec.VCpus
		pool.Status.MemoryAvailable = pool.Spec.Memory
		pool.Status.Initialized = true
	}

	promLabels := prometheus.Labels{
		"namespace": req.Namespace,
		"pool":      req.Name,
	}

	poolsMu.Lock()
	pools[poolKey] = pool

	PoolMemoryAvailable.With(promLabels).Set(float64(pool.Status.MemoryAvailable))
	PoolNetworksAvailable.With(promLabels).Set(float64(pool.Status.NetworkAvailable))
	PoolCpusAvailable.With(promLabels).Set(float64(pool.Status.VCpusAvailable))

	poolsMu.Unlock()

	reconciledPools := reconcilePoolStates()
	for _, reconciledPool := range reconciledPools {
		if reconciledPool.Name == req.Name {
			reconciledPool.Status.DeepCopyInto(&pool.Status)
			err := l.Client.Status().Update(ctx, pool)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error initializing pool status: %w", err)
			}
		}
	}

	return ctrl.Result{}, nil
}
