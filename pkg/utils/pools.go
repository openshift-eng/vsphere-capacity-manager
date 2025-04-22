package utils

import (
	"fmt"
	"math/rand"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

const (
	PoolNotSchedulable     = "Pool not schedulable"
	PoolExcluded           = "Pool marked as excluded"
	PoolNotMatchRequired   = "Pool does not match required"
	PoolInsufficientVCPU   = "Insufficient VCPU"
	PoolInsufficientMemory = "Insufficient memory"
)

type PoolFittingInfo struct {
	Pool         *v1.Pool
	MatchResults string
}

// GetFittingPools returns a list of pools that have enough resources to satisfy the resource requirements and a list of
// PoolFittingInfo specifying why pool is not a match.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func GetFittingPools(lease *v1.Lease, pools []*v1.Pool) ([]*v1.Pool, []*PoolFittingInfo) {
	var fittingPools []*v1.Pool
	poolResults := []*PoolFittingInfo{}

	for _, pool := range pools {
		if pool.Spec.NoSchedule {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolNotSchedulable})
			continue
		}
		nameMatch := len(lease.Spec.RequiredPool) > 0 && lease.Spec.RequiredPool == pool.ObjectMeta.Name
		if !nameMatch && pool.Spec.Exclude {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolExcluded})
			continue
		}
		if len(lease.Spec.RequiredPool) > 0 && !nameMatch {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolNotMatchRequired})
			continue
		}
		if int(pool.Status.VCpusAvailable) >= lease.Spec.VCpus &&
			int(pool.Status.MemoryAvailable) >= lease.Spec.Memory {
			fittingPools = append(fittingPools, pool)
		} else {
			var reason string
			if pool.Status.VCpusAvailable < lease.Spec.VCpus {
				reason = PoolInsufficientVCPU
			} else if pool.Status.MemoryAvailable < lease.Spec.Memory {
				reason = PoolInsufficientMemory
			} else {
				reason = fmt.Sprintf("[%v, %v]", PoolInsufficientVCPU, PoolInsufficientMemory)
			}

			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: reason})
		}
	}
	sort.Slice(fittingPools, func(i, j int) bool {
		iPool := fittingPools[i]
		jPool := fittingPools[j]
		cpuScoreI := float64(iPool.Status.VCpusAvailable) / float64(iPool.Spec.VCpus)
		memoryScoreI := float64(iPool.Status.MemoryAvailable) / float64(iPool.Spec.Memory)
		cpuScoreJ := float64(jPool.Status.VCpusAvailable) / float64(jPool.Spec.VCpus)
		memoryScoreJ := float64(jPool.Status.MemoryAvailable) / float64(jPool.Spec.Memory)

		return cpuScoreI+memoryScoreI > cpuScoreJ+memoryScoreJ
	})
	return fittingPools, poolResults
}

func shuffleFittingPools(pools []*v1.Pool) {
	rand.Shuffle(len(pools), func(i, j int) {
		pools[i], pools[j] = pools[j], pools[i]
	})
}

func generatePoolResults(results []*PoolFittingInfo) []string {
	var poolResults []string

	for _, result := range results {
		poolResults = append(poolResults, fmt.Sprintf("[%v: %v]", result.Pool.Name, result.MatchResults))
	}
	return poolResults
}

// GetPoolWithStrategy returns a pool that has enough resources to satisfy the lease requirements.
func GetPoolWithStrategy(lease *v1.Lease, pools []*v1.Pool, strategy v1.AllocationStrategy) (*v1.Pool, error) {
	fittingPools, results := GetFittingPools(lease, pools)

	if len(fittingPools) == 0 {
		return nil, fmt.Errorf("no pools available. %v", generatePoolResults(results))
	}
	switch strategy {
	case v1.RESOURCE_ALLOCATION_STRATEGY_RANDOM:
		shuffleFittingPools(fittingPools)
		fallthrough
	case v1.RESOURCE_ALLOCATION_STRATEGY_UNDERUTILIZED:
		fallthrough
	default:
		pool := fittingPools[0]
		lease.OwnerReferences = append(lease.OwnerReferences, metav1.OwnerReference{
			APIVersion: pool.APIVersion,
			Kind:       pool.Kind,
			Name:       pool.Name,
			UID:        pool.UID,
		})
		pool.Spec.FailureDomainSpec.DeepCopyInto(
			&lease.Status.FailureDomainSpec)

		// drop the networks from the topology. networks will be assigned in a later step.
		lease.Status.Topology.Networks = []string{}

		return pool, nil
	}
}
