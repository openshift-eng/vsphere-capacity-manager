package controller

import (
	"context"
	"fmt"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceReconciler struct {
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

func (l *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(l); err != nil {
		return fmt.Errorf("error setting up controller: %w", err)
	}

	// Set up API helpers from the manager.
	l.Client = mgr.GetClient()
	l.Scheme = mgr.GetScheme()
	l.Recorder = mgr.GetEventRecorderFor("namespaces-controller")
	l.RESTMapper = mgr.GetRESTMapper()

	return nil
}

func (l *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Printf("Reconciling namespace: %v", req)
	defer log.Print("Finished reconciling namespace")

	ns := &corev1.Namespace{}
	err := l.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: l.Namespace}, ns)

	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if ns.DeletionTimestamp == nil {
		return ctrl.Result{}, nil
	}

	var leasesToDelete []*v1.Lease

	poolsMu.Lock()
	for _, lease := range leases {
		if lease.ObjectMeta.Labels == nil {
			continue
		}
		if leaseNs, ok := lease.ObjectMeta.Labels[v1.LeaseNamespace]; ok {
			if leaseNs != req.Name {
				continue
			}
			log.Printf("lease %s is referenced by deleted namespace %s. will delete lease.", lease.Name, leaseNs)
			leasesToDelete = append(leasesToDelete, lease.DeepCopy())
		}
	}
	poolsMu.Unlock()

	for _, lease := range leasesToDelete {
		log.Printf("deleting lease %s", lease.Name)
		err = l.Client.Delete(ctx, lease)
		if err != nil {
			log.Printf("error deleting lease %s: %s", lease.Name, err)
		}
	}

	return ctrl.Result{}, nil
}
