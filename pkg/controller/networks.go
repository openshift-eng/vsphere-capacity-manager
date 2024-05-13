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

type NetworkReconciler struct {
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

func (l *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.Network{}).
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

func (l *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Print("Reconciling network")
	defer log.Print("Finished reconciling network")

	networkKey := fmt.Sprintf("%s/%s", req.Namespace, req.Name)

	// Fetch the Pool instance.
	network := &v1.Network{}
	if err := l.Get(ctx, req.NamespacedName, network); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if network.DeletionTimestamp != nil {
		log.Print("Network is being deleted")
		if network.Finalizers != nil {
			network.Finalizers = nil
			err := l.Update(ctx, network)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error updating network: %w", err)
			}
		}
		poolsMu.Lock()
		delete(pools, networkKey)
		poolsMu.Unlock()
		return ctrl.Result{}, nil
	}

	if network.Finalizers == nil {
		log.Print("setting finalizer on network")
		network.Finalizers = []string{v1.NetworkFinalizer}
		err := l.Client.Update(ctx, network)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error setting network finalizer: %w", err)
		}
	}

	poolsMu.Lock()
	networks[networkKey] = network
	poolsMu.Unlock()
	return ctrl.Result{}, nil
}
