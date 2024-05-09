package controller

import (
	"context"
	"fmt"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"strings"
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
	poolsMu.Lock()
	leases = make(map[string]*v1.Lease)
	pools = make(map[string]*v1.Pool)
	networks = make(map[string]*v1.Network)
	poolsMu.Unlock()
	return nil
}

// getAvailableNetworks retrieves networks which are not owned by a lease
func (l *LeaseReconciler) getAvailableNetworks(pool *v1.Pool) []*v1.Network {
	networksInPool := make(map[string]*v1.Network)
	availableNetworks := make([]*v1.Network, 0)
	for _, portGroupPath := range pool.Spec.Topology.Networks {
		pathParts := strings.Split(portGroupPath, "/")
		lastToken := pathParts[len(pathParts)-1]

		for _, network := range networks {
			if network.Name == lastToken {
				networksInPool[network.Name] = network
				break
			}
		}
	}

	for _, network := range networksInPool {
		hasOwner := false
		for _, lease := range leases {
			for _, ownerRef := range lease.OwnerReferences {
				if ownerRef.Name == network.Name &&
					ownerRef.Kind == network.Kind {
					hasOwner = true
					break
				}
			}
			if hasOwner {
				break
			}
		}
		if !hasOwner {
			availableNetworks = append(availableNetworks, network)
		}
	}
	return availableNetworks
}

// reconcilePoolStates updates the states of all pools. this ensures we have the most up-to-date state of the pools
// before we attempt to reconcile any leases. the pool resource statuses are not updated.
func (l *LeaseReconciler) reconcilePoolStates() []*v1.Pool {
	var outList []*v1.Pool

	for poolName, pool := range pools {
		vcpus := 0
		memory := 0
		networks := 0
		for _, lease := range leases {
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
		pool.Status.NetworkAvailable = len(pool.Spec.Topology.Networks) - networks
		pools[poolName] = pool
		outList = append(outList, pool)
	}

	return outList
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
	var err error
	log.Print("Reconciling lease")
	defer log.Print("Finished reconciling lease")

	leaseKey := fmt.Sprintf("%s/%s", req.Namespace, req.Name)
	// Fetch the Lease instance.
	lease := &v1.Lease{}
	if err := l.Get(ctx, req.NamespacedName, lease); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if lease.DeletionTimestamp != nil {
		log.Print("Lease is being deleted")
		lease.Finalizers = []string{}
		err := l.Update(ctx, lease)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error dropping finalizers from lease: %w", err)
		}
		poolsMu.Lock()
		delete(leases, leaseKey)
		l.reconcilePoolStates()
		poolsMu.Unlock()
		return ctrl.Result{}, nil
	}

	poolsMu.Lock()
	leases[leaseKey] = lease
	poolsMu.Unlock()

	if lease.Status.Phase == v1.PHASE_FULFILLED {
		log.Print("lease is already fulfilled")
		return ctrl.Result{}, nil
	}

	poolsMu.Lock()
	updatedPools := l.reconcilePoolStates()
	poolsMu.Unlock()

	lease.Status.Phase = v1.PHASE_PENDING

	pool := &v1.Pool{}
	if ref := utils.DoesLeaseHavePool(lease); ref == nil {
		pool, err = utils.GetPoolWithStrategy(lease, updatedPools, v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED)
		if err != nil {
			if l.Client.Status().Update(ctx, lease) != nil {
				log.Printf("unable to update lease: %v", err)
			}

			return ctrl.Result{}, fmt.Errorf("unable to get matching pool: %v", err)
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

	if !utils.DoesLeaseHaveNetworks(lease) {
		poolsMu.Lock()
		availableNetworks := l.getAvailableNetworks(pool)
		poolsMu.Unlock()

		if len(availableNetworks) < lease.Spec.Networks {
			return ctrl.Result{}, fmt.Errorf("lease requires %d networks, %d networks available", lease.Spec.Networks, len(availableNetworks))
		}

		for idx := 0; idx < lease.Spec.Networks; idx++ {
			network := availableNetworks[idx]
			lease.OwnerReferences = append(lease.OwnerReferences, metav1.OwnerReference{
				APIVersion: network.APIVersion,
				Kind:       network.Kind,
				Name:       network.Name,
				UID:        network.UID,
			})
		}
	}

	leaseStatus := lease.Status.DeepCopy()
	err = l.Client.Update(ctx, lease)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating lease, requeuing: %v", err)
	}

	leaseStatus.DeepCopyInto(&lease.Status)
	lease.Status.Phase = v1.PHASE_FULFILLED
	err = l.Client.Status().Update(ctx, lease)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating lease, requeuing: %v", err)
	}

	if pool.Annotations == nil {
		pool.Annotations = make(map[string]string)
	}

	err = l.bumpPool(ctx, lease)
	if err != nil {
		log.Printf("error bumping pool: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
