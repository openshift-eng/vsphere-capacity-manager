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
	PoolLabelMismatch      = "Pool labels do not match poolSelector"
	PoolTaintNotTolerated  = "Pool has taints not tolerated by lease"
)

type PoolFittingInfo struct {
	Pool         *v1.Pool
	MatchResults string
}

// tolerationMatchesTaint checks if a toleration matches a taint.
func tolerationMatchesTaint(toleration *v1.Toleration, taint *v1.Taint) bool {
	// If toleration has an effect specified, it must match the taint's effect
	if toleration.Effect != "" && toleration.Effect != string(taint.Effect) {
		return false
	}

	// Handle Exists operator - matches if key matches (or key is empty for wildcard)
	if toleration.Operator == v1.TolerationOpExists {
		// Empty key means tolerate all taints with any key
		return toleration.Key == "" || toleration.Key == taint.Key
	}

	// Default to Equal operator
	// Must match both key and value
	return toleration.Key == taint.Key && toleration.Value == taint.Value
}

// leaseToleratesPoolTaints checks if a lease has tolerations for all of a pool's taints.
// Returns true if the lease can be scheduled on the pool, false otherwise.
func leaseToleratesPoolTaints(lease *v1.Lease, pool *v1.Pool) bool {
	// If pool has no taints, lease can always be scheduled
	if len(pool.Spec.Taints) == 0 {
		return true
	}

	// Check each taint to see if it's tolerated
	for _, taint := range pool.Spec.Taints {
		tolerated := false

		// Check if any of the lease's tolerations match this taint
		for i := range lease.Spec.Tolerations {
			if tolerationMatchesTaint(&lease.Spec.Tolerations[i], &taint) {
				tolerated = true
				break
			}
		}

		// If this taint is not tolerated, the lease cannot be scheduled on this pool
		if !tolerated {
			return false
		}
	}

	// All taints are tolerated
	return true
}

// poolMatchesSelector checks if a pool's labels match the lease's poolSelector.
// Returns true if all selector labels match the pool's labels.
func poolMatchesSelector(lease *v1.Lease, pool *v1.Pool) bool {
	// If no selector is specified, pool matches
	if len(lease.Spec.PoolSelector) == 0 {
		return true
	}

	// Check that all selector key-value pairs exist in the pool's labels
	for key, value := range lease.Spec.PoolSelector {
		poolValue, exists := pool.Labels[key]
		if !exists || poolValue != value {
			return false
		}
	}

	return true
}

// GetFittingPools returns a list of pools that have enough resources to satisfy the resource requirements and a list of
// PoolFittingInfo specifying why pool is not a match.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
func GetFittingPools(lease *v1.Lease, pools []*v1.Pool) ([]*v1.Pool, []*PoolFittingInfo) {
	var fittingPools []*v1.Pool
	poolResults := []*PoolFittingInfo{}

	for _, pool := range pools {
		// Check if this pool is already owned by the lease
		alreadyOwned := false
		for _, ownerRef := range lease.OwnerReferences {
			if ownerRef.Kind == "Pool" && ownerRef.Name == pool.Name {
				alreadyOwned = true
				break
			}
		}
		if alreadyOwned {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: "Pool already assigned to lease"})
			continue
		}

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
		// Check if pool labels match the lease's poolSelector
		if !poolMatchesSelector(lease, pool) {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolLabelMismatch})
			continue
		}
		// Check if lease tolerates all pool taints
		if !leaseToleratesPoolTaints(lease, pool) {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolTaintNotTolerated})
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

		// Check if this pool is already an owner reference
		alreadyOwner := false
		for _, ref := range lease.OwnerReferences {
			if ref.Kind == "Pool" && ref.Name == pool.Name {
				alreadyOwner = true
				break
			}
		}

		if !alreadyOwner {
			lease.OwnerReferences = append(lease.OwnerReferences, metav1.OwnerReference{
				APIVersion: pool.APIVersion,
				Kind:       pool.Kind,
				Name:       pool.Name,
				UID:        pool.UID,
			})
		}

		return pool, nil
	}
}
