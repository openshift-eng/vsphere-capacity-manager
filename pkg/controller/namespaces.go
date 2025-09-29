package controller

import (
	"context"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/openshift-eng/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
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
	// Set up API helpers from the manager.
	l.Client = mgr.GetClient()
	l.Scheme = mgr.GetScheme()
	l.Recorder = mgr.GetEventRecorderFor("namespaces-controller")
	l.RESTMapper = mgr.GetRESTMapper()

	go func() {
		ctx := context.TODO()
		for {
			log.Printf("checking for abandoned leases")
			l.PruneAbandonedLeases(ctx)
			time.Sleep(5 * time.Minute)
		}
	}()
	return nil
}

func (l *NamespaceReconciler) PruneAbandonedLeases(ctx context.Context) {
	namespaces := &corev1.NamespaceList{}

	reconcileLock.Lock()
	defer reconcileLock.Unlock()

	err := l.Client.List(ctx, namespaces)
	if err != nil {
		log.Printf("Failed to list namespaces: %v", err)
		return
	}

	var leasesToDelete []*v1.Lease

	for _, lease := range leases {
		if leaseNs, ok := lease.ObjectMeta.Labels[v1.LeaseNamespace]; ok {
			nsFound := false
			for _, ns := range namespaces.Items {
				if ns.Name == leaseNs {
					nsFound = true
					break
				}
			}
			if nsFound {
				continue
			}
			log.Printf("lease %s is referenced by deleted namespace %s. will delete lease.", lease.Name, leaseNs)
			leasesToDelete = append(leasesToDelete, lease.DeepCopy())
		}
	}

	for _, lease := range leasesToDelete {
		log.Printf("deleting lease %s", lease.Name)
		err = l.Client.Delete(ctx, lease)
		if err != nil {
			log.Printf("error deleting lease %s: %s", lease.Name, err)
		}
	}
}
