package test

import (
	"context"
	"fmt"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getPools returns a list of pools for testing
func getPools() *v1.PoolList {
	return &v1.PoolList{
		Items: []v1.Pool{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-pool-0",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      10,
					Memory:     100,
					Storage:    1000,
					Server:     "vcenter-0",
					Datacenter: "dc-0",
					Cluster:    "cluster-0",
					Datastore:  "datastore-0",
					Exclude:    false,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-pool-1",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      20,
					Memory:     200,
					Storage:    2000,
					Server:     "vcenter-1",
					Datacenter: "dc-1",
					Cluster:    "cluster-1",
					Datastore:  "datastore-1",
					Exclude:    false,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-zonal-pool-0",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      10,
					Memory:     100,
					Storage:    100,
					Server:     "vcenter-2",
					Datacenter: "dc-2",
					Cluster:    "cluster-2",
					Datastore:  "datastore-2",
					Exclude:    true,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample-zonal-pool-1",
					Namespace: "default",
				},
				Spec: v1.PoolSpec{
					VCpus:      10,
					Memory:     100,
					Storage:    100,
					Server:     "vcenter-3",
					Datacenter: "dc-3",
					Cluster:    "cluster-3",
					Datastore:  "datastore-3",
					Exclude:    true,
				},
			},
		},
	}
}

type shape int64

const (
	SHAPE_SMALL  = shape(1)
	SHAPE_MEDIUM = shape(10)
	SHAPE_LARGE  = shape(100)
)

type resourceRequest struct {
	request v1.ResourceRequest
}

// getResourceRequest returns a ResourceRequest object for testing
func GetResourceRequest() *resourceRequest {
	return &resourceRequest{
		request: v1.ResourceRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample-request-0",
				Namespace: "default",
			},
		},
	}
}

func (r *resourceRequest) WithName(name string) *resourceRequest {
	r.request.ObjectMeta.Name = name
	return r
}

func (r *resourceRequest) WithPoolCount(cnt int) *resourceRequest {
	r.request.Spec.VCenters = cnt
	return r
}

func (r *resourceRequest) WithShape(shape shape) *resourceRequest {
	r.request.Spec.VCpus = int(16 * int64(shape))
	r.request.Spec.Memory = int(16 * int64(shape))
	r.request.Spec.Storage = int(120 * int64(shape))
	r.request.Spec.Networks = int(1 * int64(shape))

	return r
}

func (r *resourceRequest) WithPool(pool string) *resourceRequest {
	r.request.Spec.RequiredPool = pool
	return r
}

func (r *resourceRequest) Build() *v1.ResourceRequest {
	if r.request.Spec.VCenters == 0 {
		r.request.Spec.VCenters = 1
	}
	return &r.request
}

// IsLeaseReflectedInPool checks if the lease is reflected in the pool
func IsLeaseReflectedInPool(ctx context.Context, client client.Client, lease *v1.Lease) (bool, error) {
	if lease.Status.Pool == nil {
		return false, fmt.Errorf("lease %s does not have an associated pool", lease.Name)
	}
	pool := &v1.Pool{}
	if err := client.Get(ctx, types.NamespacedName{Name: lease.Status.Pool.Name, Namespace: lease.Namespace}, pool); err != nil {
		return false, fmt.Errorf("error getting pool: %w", err)
	}
	for _, ref := range pool.Status.Leases {
		if ref.Name == lease.Name {
			if pool.Status.VCpusAvailable == pool.Spec.VCpus-lease.Spec.VCpus ||
				pool.Status.MemoryAvailable == pool.Spec.Memory-lease.Spec.Memory ||
				pool.Status.DatastoreAvailable == pool.Spec.Storage-lease.Spec.Storage ||
				pool.Status.NetworkAvailable == pool.Status.NetworkAvailable-len(lease.Status.PortGroups) {
				return true, nil
			}
		}
	}
	return false, nil
}
