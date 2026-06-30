package controller

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	generator "github.com/docker/docker/pkg/namesgenerator"
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

	reconcileLock.Lock()
	defer reconcileLock.Unlock()

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
		delete(pools, poolKey)
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

	poolUpdateNeeded := false
	// Set ShortName for pools that do not have it set.  This should only be older configs.
	if len(pool.Spec.ShortName) == 0 {
		// TODO - Need to enhance this logic to make sure generator does not come up w/ a name that is already in use
		//        There has been a case where two pools ended up w/ same shortName.
		pool.Spec.ShortName = strings.ReplaceAll(generator.GetRandomName(0), "_", "-")
		log.Printf("Setting ShortName for pool %v to %v\n", pool.Name, pool.Spec.ShortName)
		poolUpdateNeeded = true
	}

	// We are doing this separate from the call above for now due to existing pools may have short names with this invalid
	// character in it.  This will cause issues in the installer if zone name used in control plane / compute pool zone
	// configuration.
	if strings.Contains(pool.Spec.ShortName, "_") {
		pool.Spec.ShortName = strings.ReplaceAll(pool.Spec.ShortName, "_", "-")
		log.Printf("Updating ShortName for pool %v to %v\n", pool.Name, pool.Spec.ShortName)
		poolUpdateNeeded = true
	}

	// Moved update out of above info to reduce updates.
	if poolUpdateNeeded {
		err := l.Client.Update(ctx, pool)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error setting pool short name: %w", err)
		}
	}

	if !pool.Status.Initialized {
		pool.Status.VCpusAvailable = pool.Spec.VCpus
		pool.Status.MemoryAvailable = pool.Spec.Memory
		pool.Status.Initialized = true
	}

	pools[poolKey] = pool

	reconciledPools := reconcilePoolStates()
	for _, reconciledPool := range reconciledPools {
		if reconciledPool.Name == req.Name {
			reconciledPool.Status.DeepCopyInto(&pool.Status)
			err := l.Client.Status().Update(ctx, pool)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error updating pool status: %w", err)
			}
		}
	}

	promLabels := prometheus.Labels{
		"namespace": req.Namespace,
		"pool":      req.Name,
	}

	PoolMemoryAvailable.With(promLabels).Set(float64(pool.Status.MemoryAvailable))
	PoolMemoryTotal.With(promLabels).Set(float64(pool.Spec.Memory))
	PoolNetworksAvailable.With(promLabels).Set(float64(pool.Status.NetworkAvailable))
	PoolNetworksTotal.With(promLabels).Set(float64(len(pool.Spec.Topology.Networks)))
	PoolCpusAvailable.With(promLabels).Set(float64(pool.Status.VCpusAvailable))
	PoolCpusTotal.With(promLabels).Set(float64(pool.Spec.VCpus))
	LeasesInUse.With(promLabels).Set(float64(pool.Status.LeaseCount))

	overCommitRatio, err := strconv.ParseFloat(pool.Spec.OverCommitRatio, 64)
	if err != nil {
		overCommitRatio = 1.0
	}
	effectiveCpus := float64(pool.Spec.VCpus) * overCommitRatio
	if effectiveCpus > 0 {
		PoolVcpusUtilizationRatio.With(promLabels).Set((effectiveCpus - float64(pool.Status.VCpusAvailable)) / effectiveCpus)
	}
	if pool.Spec.Memory > 0 {
		PoolMemoryUtilizationRatio.With(promLabels).Set(float64(pool.Spec.Memory-pool.Status.MemoryAvailable) / float64(pool.Spec.Memory))
	}
	networksTotal := float64(len(pool.Spec.Topology.Networks))
	if networksTotal > 0 {
		PoolNetworksUtilizationRatio.With(promLabels).Set((networksTotal - float64(pool.Status.NetworkAvailable)) / networksTotal)
	}

	noSchedule := float64(0)
	if pool.Spec.NoSchedule {
		noSchedule = 1
	}
	PoolNoSchedule.With(promLabels).Set(noSchedule)

	excluded := float64(0)
	if pool.Spec.Exclude {
		excluded = 1
	}
	PoolExcluded.With(promLabels).Set(excluded)

	return ctrl.Result{}, nil
}
