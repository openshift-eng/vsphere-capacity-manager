package utils

import (
	"fmt"
	"math/rand"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis/vspherecapacitymanager.splat.io/v1"
)

const (
	PoolNotSchedulable      = "Pool not schedulable"
	PoolExcluded            = "Pool marked as excluded"
	PoolNotMatchRequired    = "Pool does not match required"
	PoolInsufficientVCPU    = "Insufficient VCPU"
	PoolInsufficientMemory  = "Insufficient memory"
	PoolLabelMismatch       = "Pool labels do not match poolSelector"
	PoolTaintNotTolerated   = "Pool has taints not tolerated by lease"
	PoolVCenterLimitReached = "Pool vCenter limit reached"
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

// LeaseToleratesPoolTaints checks if a lease has tolerations for all of a pool's taints.
// Returns true if the lease can be scheduled on the pool, false otherwise.
func LeaseToleratesPoolTaints(lease *v1.Lease, pool *v1.Pool) bool {
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

// PoolMatchesSelector checks if a pool's labels match the lease's poolSelector.
// Returns true if all selector labels match the pool's labels.
func PoolMatchesSelector(lease *v1.Lease, pool *v1.Pool) bool {
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

// GetVCentersInUse returns the set of distinct vCenter Server FQDNs already used
// by the given pools (e.g., a lease's currently-assigned pools).
func GetVCentersInUse(assignedPools []*v1.Pool) map[string]bool {
	vcenters := make(map[string]bool)
	for _, pool := range assignedPools {
		if pool.Spec.Server != "" {
			vcenters[pool.Spec.Server] = true
		}
	}
	return vcenters
}

// poolMatchesStructural reports whether a pool structurally matches a lease's requirements,
// ignoring transient state such as current resource availability, current ownership, and
// any vCenter exclusions computed for a particular reconcile pass. It captures only the
// checks that depend on static configuration (RequiredPool/Exclude, PoolSelector,
// Tolerations, NoSchedule), which is what determines whether a request could ever be
// satisfied by this pool, as opposed to whether it can be satisfied right now.
func poolMatchesStructural(lease *v1.Lease, pool *v1.Pool) bool {
	if pool.Spec.NoSchedule {
		return false
	}
	nameMatch := len(lease.Spec.RequiredPool) > 0 && lease.Spec.RequiredPool == pool.ObjectMeta.Name
	if !nameMatch && pool.Spec.Exclude {
		return false
	}
	if len(lease.Spec.RequiredPool) > 0 && !nameMatch {
		return false
	}
	if !PoolMatchesSelector(lease, pool) {
		return false
	}
	if !LeaseToleratesPoolTaints(lease, pool) {
		return false
	}
	return true
}

// MaxAchievablePools returns the maximum number of pools a lease could ever be assigned,
// given the full known pool inventory and the lease's structural constraints (RequiredPool,
// PoolSelector, Tolerations, NoSchedule/Exclude) and its VCenters cap. It deliberately
// ignores current resource availability and existing ownership, since those are transient:
// this answers "can this request ever be satisfied," not "can it be satisfied right now."
func MaxAchievablePools(lease *v1.Lease, allPools []*v1.Pool) int {
	poolsPerVCenter := make(map[string]int)
	for _, pool := range allPools {
		if !poolMatchesStructural(lease, pool) {
			continue
		}
		poolsPerVCenter[pool.Spec.Server]++
	}

	if lease.Spec.VCenters <= 0 {
		total := 0
		for _, count := range poolsPerVCenter {
			total += count
		}
		return total
	}

	counts := make([]int, 0, len(poolsPerVCenter))
	for _, count := range poolsPerVCenter {
		counts = append(counts, count)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(counts)))

	n := lease.Spec.VCenters
	if n > len(counts) {
		n = len(counts)
	}
	total := 0
	for i := 0; i < n; i++ {
		total += counts[i]
	}
	return total
}

// IsLeaseSatisfiable reports whether a lease's request could ever be fulfilled given the
// full known pool inventory. It returns false with an explanatory reason when the lease's
// Pools/VCenters requirements structurally exceed what the known inventory can ever provide,
// regardless of current utilization by other leases.
func IsLeaseSatisfiable(lease *v1.Lease, allPools []*v1.Pool) (bool, string) {
	requiredPools := lease.Spec.Pools
	if requiredPools == 0 {
		requiredPools = 1
	}

	maxAchievable := MaxAchievablePools(lease, allPools)
	if maxAchievable >= requiredPools {
		return true, ""
	}

	if reason, ok := requiredPoolUnsatisfiableReason(lease, allPools); ok {
		return false, reason
	}

	return false, fmt.Sprintf(
		"lease requires %d pool(s) (vcenters cap %d) but at most %d are structurally achievable given the known pool inventory",
		requiredPools, lease.Spec.VCenters, maxAchievable,
	)
}

// requiredPoolUnsatisfiableReason returns a specific, actionable reason when a lease's
// RequiredPool can never be scheduled — either because no pool with that name exists, or
// because the named pool is disabled (NoSchedule). It returns ("", false) when RequiredPool
// isn't set, or when the named pool exists and is schedulable, in which case any
// unsatisfiability comes from some other structural mismatch (e.g. PoolSelector,
// Tolerations) and the caller should fall back to its generic reason.
func requiredPoolUnsatisfiableReason(lease *v1.Lease, allPools []*v1.Pool) (string, bool) {
	if lease.Spec.RequiredPool == "" {
		return "", false
	}

	for _, pool := range allPools {
		if pool.ObjectMeta.Name != lease.Spec.RequiredPool {
			continue
		}
		if pool.Spec.NoSchedule {
			return fmt.Sprintf("required pool %q is disabled and cannot be scheduled", lease.Spec.RequiredPool), true
		}
		return "", false
	}

	return fmt.Sprintf("required pool %q does not exist", lease.Spec.RequiredPool), true
}

// GetFittingPools returns a list of pools that have enough resources to satisfy the resource requirements and a list of
// PoolFittingInfo specifying why pool is not a match.
// The list is sorted by the sum of the resource usage of the pool. The pool with the least resource usage is first.
// excludedVCenters is an optional set of vCenter Server FQDNs to exclude from consideration (used to enforce
// the lease's VCenters cap). Pass nil or an empty map for no vcenter constraint.
func GetFittingPools(lease *v1.Lease, pools []*v1.Pool, excludedVCenters map[string]bool) ([]*v1.Pool, []*PoolFittingInfo) {
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
		if !PoolMatchesSelector(lease, pool) {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolLabelMismatch})
			continue
		}
		// Check if lease tolerates all pool taints
		if !LeaseToleratesPoolTaints(lease, pool) {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolTaintNotTolerated})
			continue
		}

		// If a vCenter cap is in effect, skip pools on excluded vCenters
		// This check comes after fundamental pool properties and lease requirements
		// to ensure more specific rejection reasons are reported first
		if len(excludedVCenters) > 0 && excludedVCenters[pool.Spec.Server] {
			poolResults = append(poolResults, &PoolFittingInfo{Pool: pool, MatchResults: PoolVCenterLimitReached})
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
// excludedVCenters is an optional set of vCenter Server FQDNs to exclude (enforces the VCenters cap).
// Pass nil for no vcenter constraint.
func GetPoolWithStrategy(lease *v1.Lease, pools []*v1.Pool, strategy v1.AllocationStrategy, excludedVCenters map[string]bool) (*v1.Pool, error) {
	fittingPools, results := GetFittingPools(lease, pools, excludedVCenters)

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
