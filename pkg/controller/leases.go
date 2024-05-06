package controller

import (
	"context"
	"fmt"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/utils"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// reconcilePoolStates updates the states of all pools. this ensures we have the most up-to-date state of the pools
// before we attempt to reconcile any leases. the pool resource statuses are not updated.
func (l *LeaseReconciler) reconcilePoolStates(ctx context.Context, req ctrl.Request) ([]*v1.Pool, error) {
	pools := v1.PoolList{}
	err := l.Client.List(ctx, &pools)
	if err != nil {
		return nil, fmt.Errorf("error listing pools: %w", err)
	}

	var outList []*v1.Pool
	var leases v1.LeaseList
	err = l.Client.List(ctx, &leases, client.InNamespace(req.Namespace))
	if err != nil {
		return nil, fmt.Errorf("error listing leases: %w", err)
	}

	for idx, pool := range pools.Items {
		vcpus := 0
		memory := 0
		networks := 0
		for _, lease := range leases.Items {
			if lease.Status.Phase != v1.PHASE_FULFILLED {
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
		outList = append(outList, &pools.Items[idx])
	}

	return outList, nil
}

func (l *LeaseReconciler) bumpPool(ctx context.Context, lease *v1.Lease) error {
	pool := &v1.Pool{}
	for _, ownerRef := range lease.OwnerReferences {
		if ownerRef.Kind == "Pool" {
			err := l.Get(ctx, types.NamespacedName{
				Name:      ownerRef.Name,
				Namespace: lease.Namespace,
			}, pool)
			if err != nil {
				log.Printf("error getting pool object: %v", err)
				return err
			}
			break
		}
	}
	if pool.Annotations == nil {
		pool.Annotations = make(map[string]string)
	}
	pool.Annotations["last-updated"] = time.Now().Format(time.RFC3339)
	err := l.Client.Update(ctx, pool)
	if err != nil {
		return fmt.Errorf("error updating pool, requeuing: %v", err)
	}
	return nil
}

func (l *LeaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Print("Reconciling lease")
	defer log.Print("Finished reconciling lease")

	// Fetch the Lease instance.
	lease := &v1.Lease{}
	if err := l.Get(ctx, req.NamespacedName, lease); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if lease.DeletionTimestamp != nil {
		log.Print("Lease is being deleted")
		err := l.bumpPool(ctx, lease)
		if err != nil {
			log.Printf("error updating pool: %v", err)
			return ctrl.Result{}, err
		}

		lease.Finalizers = []string{}
		err = l.Update(ctx, lease)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating lease: %w", err)
		}
		return ctrl.Result{}, err
	}

	if lease.Status.Phase == v1.PHASE_FULFILLED {
		log.Print("lease is already fulfilled")
		return ctrl.Result{}, nil
	}

	pools := v1.PoolList{}
	err := l.Client.List(ctx, &pools)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing pools, requeuing: %v", err)
	}

	updatedPools, err := l.reconcilePoolStates(ctx, req)
	if err != nil {

		return ctrl.Result{}, fmt.Errorf("error updating pool states, requeuing: %v", err)
	}

	lease.Status.Phase = v1.PHASE_PENDING

	var pool *v1.Pool
	if ref := utils.DoesLeaseHavePool(lease); ref == nil {
		pool, err = utils.GetPoolWithStrategy(lease, updatedPools, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
		if err != nil {
			if l.Client.Status().Update(ctx, lease) != nil {
				log.Printf("unable to update lease: %v", err)
			}

			return ctrl.Result{
				RequeueAfter: 5 * time.Second,
			}, fmt.Errorf("unable to get matching pool: %v", err)
		}
	} else {
		err = l.Get(ctx, types.NamespacedName{
			Namespace: req.Namespace,
			Name:      ref.Name,
		}, pool)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error getting pool: %v", err)
		}
	}
	err = l.Client.Update(ctx, lease)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating lease, requeuing: %v", err)
	}
	lease.Status.Phase = v1.PHASE_FULFILLED
	err = l.Client.Status().Update(ctx, lease)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating lease, requeuing: %v", err)
	}

	if pool.Annotations == nil {
		pool.Annotations = make(map[string]string)
	}
	pool.Annotations["last-updated"] = time.Now().Format(time.RFC3339)
	err = l.Client.Update(ctx, pool)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating pool, requeuing: %v", err)
	}
	return ctrl.Result{}, nil
}
