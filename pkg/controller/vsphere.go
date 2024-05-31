package controller

import (
	"context"
	"fmt"
	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	"github.com/openshift-splat-team/vsphere-capacity-manager/pkg/utils"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"log"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

type poolClient struct {
	client     *vim25.Client
	restClient *rest.Client
	logoutFunc utils.ClientLogout
}

type VSphereReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
	RESTMapper     meta.RESTMapper
	UncachedClient client.Client

	Namespace string

	// OperatorName is the name of the ClusterOperator with which the controller should report
	// its status.
	OperatorName string

	// ReleaseVersion is the version of current cluster operator release.
	ReleaseVersion string

	ClientCache map[string]*poolClient
}

func (v *VSphereReconciler) getClient(ctx context.Context, pool *v1.Pool) (*vim25.Client, *rest.Client, utils.ClientLogout, error) {
	if poolClient, exists := v.ClientCache[pool.Name]; exists {
		return poolClient.client, poolClient.restClient, poolClient.logoutFunc, nil
	}

	annotations := pool.Annotations
	if annotations == nil {
		return nil, nil, nil, fmt.Errorf("no annotations found in pool, could not derive auth path for pool")
	}

	if _, exists := annotations[v1.PoolAuthPath]; !exists {
		return nil, nil, nil, fmt.Errorf("annotation %s not found, could not derive auth path for pool", v1.PoolAuthPath)
	}

	authPath := annotations[v1.PoolAuthPath]
	credentials, err := utils.LoadCredentialsFromPath(authPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not load credentials from path %s: %w", authPath, err)
	}

	randomCredIdx := rand.Int() % len(credentials)
	idx := 0
	var username, password string
	for username, password = range credentials {
		if idx == randomCredIdx {
			break
		}
		idx++
	}
	return utils.CreateVSphereClients(ctx, pool.Spec.Server, username, password)
}

func (v *VSphereReconciler) hasMaintenanceMode(ctx context.Context, pool *v1.Pool) (bool, error) {
	vClient, _, _, err := v.getClient(ctx, pool)
	if err != nil {
		return false, fmt.Errorf("unable to get client: %v", err)
	}
	finder := find.NewFinder(vClient, false)
	clusterRef, err := finder.ClusterComputeResource(ctx, pool.Spec.Topology.ComputeCluster)
	if err != nil {
		return false, fmt.Errorf("unable to get cluster reference: %w", err)
	}

	var hostSystemManagedObject mo.HostSystem
	hosts, err := clusterRef.Hosts(ctx)

	for _, hostObj := range hosts {
		err := hostObj.Properties(ctx, hostObj.Reference(), []string{"config.product", "network", "datastore", "runtime"}, &hostSystemManagedObject)
		if err != nil {
			return false, fmt.Errorf("unable to get host system managed object: %w", err)
		}
		if hostSystemManagedObject.Runtime.InMaintenanceMode {
			return true, nil
		}
	}

	return false, nil
}

func (v *VSphereReconciler) Reconcile() {
	ctx := context.Background()

	poolsToCheck := make([]*v1.Pool, 0)

	poolsMu.Lock()
	for _, pool := range pools {
		if pool.Spec.NoSchedule {
			continue
		}
		if pool.Annotations != nil {
			if _, exists := pool.Annotations[v1.PoolDisablePoolChecks]; exists {
				log.Printf("pool %s has pool checks disabled", pool.Name)
				continue
			}
		}
		poolCopy := pool.DeepCopy()
		poolsToCheck = append(poolsToCheck, poolCopy)
	}
	poolsMu.Unlock()

	timedCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	for _, pool := range poolsToCheck {
		maintenanceMode, err := v.hasMaintenanceMode(timedCtx, pool)
		if err != nil {
			log.Printf("unable to check maintenance mode for pool %s: %v", pool.Name, err)
		}
		if maintenanceMode {
			if pool.Spec.NoSchedule {
				continue
			}
			err := v.Client.Get(ctx, client.ObjectKeyFromObject(pool), pool)
			if err != nil {
				log.Printf("unable to get pool %s: %v", pool.Name, err)
				return
			}

			log.Printf("pool  %s has atleast one host in maintenance mode. disabling scheduling for this pool.", pool.Name)

			pool.Status.Degraded = true
			err = v.Client.Status().Update(ctx, pool)
			if err != nil {
				log.Printf("unable to update pool status %s: %v", pool.Name, err)
				continue
			}
			pool.Spec.NoSchedule = true
			err = v.Client.Update(ctx, pool)
			if err != nil {
				log.Printf("unable to update pool %s: %v", pool.Name, err)
				continue
			}
		} else if pool.Status.Degraded {
			err := v.Client.Get(ctx, client.ObjectKeyFromObject(pool), pool)
			if err != nil {
				log.Printf("unable to get pool %s: %v", pool.Name, err)
				return
			}
			pool.Status.Degraded = false

			err = v.Client.Status().Update(ctx, pool)
			if err != nil {
				log.Printf("unable to update pool status %s: %v", pool.Name, err)
				continue
			}
		}
	}
}

func (v *VSphereReconciler) setupTimer() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		v.Reconcile()
	}
}

func (v *VSphereReconciler) SetupWithManager(mgr manager.Manager) error {
	v.Client = mgr.GetClient()
	go v.setupTimer()
	return nil
}
