package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

func (l *LeaseReconciler) setLeasePhase(ctx context.Context, lease *v1.Lease, phase v1.Phase) error {
	lease.Status.Phase = phase
	return l.Client.Status().Update(ctx, lease)
}

func (l *LeaseReconciler) ensureLeaseIsRemovedFromPool(ctx context.Context, lease *v1.Lease) (ctrl.Result, error) {
	var pool *v1.Pool = &v1.Pool{}
	err := l.Client.Get(ctx, types.NamespacedName{
		Namespace: lease.Namespace,
		Name:      lease.Status.Pool.Name,
	}, pool)
	if err != nil {
		log.Printf("error getting pool, requeuing: %v", err)
		return ctrl.Result{
			RequeueAfter: 2 * time.Second,
		}, nil
	}

	// attempt to unbind the lease from the pool active port groups
	adjustedActivePortGroups := []v1.Network{}
	for _, portGroup := range pool.Status.ActivePortGroups {
		found := false
		for _, leasePortGroup := range lease.Status.PortGroups {
			if resources.CompareNetworks(portGroup, leasePortGroup) {
				found = true
				break
			}
		}
		if !found {
			adjustedActivePortGroups = append(adjustedActivePortGroups, portGroup)
		}
	}

	pool.Status.ActivePortGroups = adjustedActivePortGroups

	// attempt to unbind the lease from the pool leases
	adjustedLeases := []*corev1.TypedLocalObjectReference{}
	for _, poolLease := range pool.Status.Leases {
		if poolLease.Name != lease.Name {
			adjustedLeases = append(adjustedLeases, poolLease)
		}
	}
	pool.Status.Leases = adjustedLeases

	pool.Status.VCpusAvailable += lease.Spec.VCpus
	pool.Status.MemoryAvailable += lease.Spec.Memory
	pool.Status.DatastoreAvailable += lease.Spec.Storage
	pool.Status.NetworkAvailable = len(pool.Status.PortGroups) - len(pool.Status.ActivePortGroups)

	err = l.Client.Status().Update(ctx, pool)
	if err != nil {
		log.Printf("error updating pool, requeuing: %v", err)
		return ctrl.Result{
			RequeueAfter: 2 * time.Second,
		}, nil
	}
	return ctrl.Result{}, nil
}

func (l *LeaseReconciler) ensureLeaseIsInPool(ctx context.Context, lease *v1.Lease) (ctrl.Result, error) {
	var pool *v1.Pool = &v1.Pool{}

	err := l.Client.Get(ctx, types.NamespacedName{
		Namespace: lease.Namespace,
		Name:      lease.Status.Pool.Name,
	}, pool)
	if err != nil {
		log.Printf("error getting pool, requeuing: %v", err)
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	for _, poolLease := range pool.Status.Leases {
		if poolLease.Name == lease.Name {
			err = l.setLeasePhase(ctx, lease, v1.PHASE_FULFILLED)
			if err != nil {
				log.Printf("error setting lease phase, requeuing: %v", err)
				return ctrl.Result{
					RequeueAfter: 5 * time.Second,
				}, nil
			}
			return ctrl.Result{}, nil
		}
	}

	pool.Status.Leases = append(pool.Status.Leases, &corev1.TypedLocalObjectReference{
		Name:     lease.Name,
		APIGroup: &v1.GroupVersion.Group,
		Kind:     "Lease",
	})

	resources.CalculateResourceUsage(pool, lease)

	log.Printf("updating pool with lease: leases %d", len(pool.Status.Leases))
	err = l.Client.Status().Update(ctx, pool)
	if err != nil {
		log.Printf("error updating pool, requeuing: %v", err)
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	lease.Status.Pool = &corev1.TypedLocalObjectReference{
		Name: pool.Name,
	}

	err = l.setLeasePhase(ctx, lease, v1.PHASE_FULFILLED)
	if err != nil {
		log.Printf("error setting lease phase, requeuing: %v", err)
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (l *LeaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Print("Reconciling lease")
	defer log.Print("Finished reconciling lease")

	// Fetch the Lease instance.
	lease := &v1.Lease{}
	if err := l.Get(ctx, req.NamespacedName, lease); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pools := v1.PoolList{}
	err := l.Client.List(ctx, &pools)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error listing pools: %w", err)
	}

	if lease.DeletionTimestamp != nil {
		log.Print("Lease is being deleted")
		return l.ensureLeaseIsRemovedFromPool(ctx, lease)
	}

	if lease.Status.Phase == v1.PHASE_FULFILLED {
		log.Print("lease is already fulfilled")
		return ctrl.Result{}, nil

	}
	if lease.Status.Pool != nil {
		log.Print("lease already has a pool")
		return ctrl.Result{}, nil
	}

	lease.Status.Phase = v1.PHASE_PENDING

	pool, err := resources.GetPoolWithStrategy(lease, pools.Items, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
	if err != nil {
		l.Client.Status().Update(ctx, lease)
		log.Printf("error getting pool: %v", err)
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	lease.Status.Pool = &corev1.TypedLocalObjectReference{
		Name: pool.Name,
	}

	err = l.Client.Get(ctx, client.ObjectKeyFromObject(pool), pool)
	if err != nil {
		log.Printf("error getting pool: %v", err)
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	log.Printf("Updating pool %s with last lease update annotation %v", pool.Name, &pool.ObjectMeta.Annotations)
	err = l.Client.Update(ctx, pool)
	if err != nil {
		log.Printf("error udpating pool: %v", err)
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	err = l.Client.Status().Update(ctx, lease)
	if err != nil {
		log.Printf("error updating lease: %v", err)
		return ctrl.Result{
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	return l.ensureLeaseIsInPool(ctx, lease)
}
