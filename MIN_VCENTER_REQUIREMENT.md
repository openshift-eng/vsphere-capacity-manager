# Minimum vCenter Requirement (`spec.min-vcenters`)

## Overview

`spec.min-vcenters` allows a single multi-pool lease to require that its assigned
pools span at least N distinct vCenters (identified by Pool `spec.server` FQDN).
This lets a CI job request all of its desired pools through one lease instead of
creating several leases and stitching the results together, while still
guaranteeing vCenter-level failure isolation.

Example — the motivating use case:

```yaml
apiVersion: vspherecapacitymanager.splat.io/v1
kind: Lease
metadata:
  name: multi-vcenter-lease
spec:
  pools: 3          # request three pools
  min-vcenters: 2   # ... spanning at least two unique vCenters
  vcpus: 28
  memory: 112
  networks: 1
```

The lease is only `Fulfilled` once it owns 3 pools drawn from at least 2
distinct vCenters. Without `min-vcenters`, the scheduler would happily assign
all 3 pools from a single vCenter when that is the most under-utilized option.

## Semantics

| Field | Meaning | Default |
|---|---|---|
| `spec.pools` | Number of pools to assign | `1` |
| `spec.vcenters` | **Maximum** number of distinct vCenters (cap) | `0` (no limit) |
| `spec.min-vcenters` | **Minimum** number of distinct vCenters required | `0` (no minimum) |

Validation and interactions (enforced by the satisfiability check, see below):

- `min-vcenters` must not exceed `pools` — each pool resides on exactly one
  vCenter, so a lease requesting 1 pool can never span 2 vCenters.
- When `vcenters` (the cap) is set, `min-vcenters` must not exceed it.
  Setting both to the same value requests an exact vCenter count.
- The pool inventory must contain at least `min-vcenters` distinct vCenters
  with pools that structurally match the lease (`required-pool`, `poolSelector`,
  tolerations, `exclude`/`noSchedule`).

Leases that violate these constraints are transitioned to `Failed` with an
`Unschedulable` reason rather than waiting forever.

## Admission-time validation (CEL)

The two spec-internal contradictions above are additionally rejected by the
API server before the lease is ever stored, via CEL validation rules
(`x-kubernetes-validations`) generated onto the Lease CRD from the
`+kubebuilder:validation:XValidation` markers on `LeaseSpec`:

- `min-vcenters` must not exceed `pools`
- `min-vcenters` must not exceed `vcenters` (only when the cap is set; an
  unset `vcenters` means no limit)

Attempting to create or update such a lease fails immediately, e.g.:

```
Lease.vspherecapacitymanager.splat.io "my-lease" is invalid:
spec: Invalid value: "object": min-vcenters must not exceed pools: each pool resides on exactly one vCenter
```

Notes:

- CEL cannot reference kebab-case property names such as `min-vcenters`
  directly, so the rules use Kubernetes' documented escaping and address the
  field as `min__dash__vcenters` (a dash becomes `__dash__`).
- CRD validation rules require Kubernetes 1.25+ (enabled by default; GA in
  1.29). The project's envtest/CI baseline is 1.29. On older clusters the
  rules are not evaluated, and the controller-side satisfiability check still
  fails such leases after admission.
- The controller-side `IsLeaseSatisfiable` checks are kept as defense in depth:
  they cover objects created before the CRD carried the CEL rules, and the
  inventory-based conditions (`required-pool`, `poolSelector`, matching vCenter
  count) that CEL cannot express.

## Selection algorithm

The lease reconciler assigns pools one at a time inside its assignment loop.
While the pools assigned so far span fewer vCenters than `min-vcenters`
("diversification pending"):

1. **Forced diversification** — pool selection excludes every vCenter already
   in use by the lease (`utils.GetDiversityExcludedVCenters`). The next pool
   must therefore come from a new vCenter. This makes it impossible for the
   lease to paint itself into a corner by filling all of its pool slots from
   fewer vCenters than required.
2. **Cap heuristics suspended** — the `vcenters`-cap dynamic-filtering
   heuristics are skipped while diversification is pending. Those heuristics
   concentrate assignments onto fewer vCenters, which directly conflicts with
   the required spreading and can exclude the very vCenters the lease must
   diversify onto. The hard "cap reached" rule cannot trigger while
   diversifying because `len(vcentersInUse) < min-vcenters <= vcenters`.

Once the minimum is met, selection (and any cap enforcement) proceeds exactly
as before: the most under-utilized fitting pool wins, subject to the
`vcenters` cap.

Rejected pools are reported with the `Pool vCenter diversity required` reason
in the pool-fitting diagnostics that surface in the lease's `Fulfilled`
condition message.

## Phase behavior

| Situation | Phase |
|---|---|
| Fewer pools than requested, or minimum not yet met but structurally possible | `Partial` — retry every 30s while waiting for capacity on a new vCenter |
| Spec contradiction (`min-vcenters` > `pools` or > `vcenters`) | Rejected by the API server at admission (CEL); older objects are `Failed` with `Unschedulable` reason |
| Minimum can never be met by the inventory (`required-pool`, `poolSelector`, too few matching vCenters) | `Failed` with `Unschedulable` reason |
| All pools assigned, spanning >= `min-vcenters`, networks allocated | `Fulfilled` |

While waiting, the `Partial` condition message reports progress, e.g.
`lease is currently assigned 1 of 3 pools, spanning 1 of 2 required vCenters`.

A `Partial` lease keeps the pools it already holds; it does not release them,
because any vCenter shortage is transient (another lease may free capacity).
This mirrors the existing behavior for plain resource shortages.

## Relationship to `spec.vcenters` (maximum)

The two fields are independent and compose:

- `pools: 4, vcenters: 3` — 4 pools from at most 3 vCenters (existing behavior).
- `pools: 3, min-vcenters: 2` — 3 pools from at least 2 vCenters (new).
- `pools: 4, vcenters: 3, min-vcenters: 2` — 4 pools from 2–3 vCenters.
- `pools: 4, vcenters: 2, min-vcenters: 2` — 4 pools from exactly 2 vCenters.

## Implementation

- API field: `LeaseSpec.MinVCenters` (`pkg/apis/vspherecapacitymanager.splat.io/v1/leases_types.go`)
- Forced-diversification exclusion set: `utils.GetDiversityExcludedVCenters` (`pkg/utils/pools.go`)
- Structural vCenter counting: `utils.CountDistinctVCenters` (`pkg/utils/pools.go`)
- Satisfiability reasons: `minVCentersUnsatisfiableReason` within `utils.IsLeaseSatisfiable` (`pkg/utils/pools.go`)
- Reconciler enforcement: assignment loop in `LeaseReconciler.Reconcile` (`pkg/controller/leases.go`)

## Testing

- Unit tests: `TestIsLeaseSatisfiableMinVCenters`, `TestGetDiversityExcludedVCenters`,
  `TestCountDistinctVCenters`, `TestGetFittingPoolsWithMinVCenterDiversity`,
  `TestGetPoolWithStrategyMinVCenterDiversity{,Unavailable}` (`pkg/utils/pools_test.go`)
- Integration tests (envtest): "should acquire lease with multiple pools across
  multiple vcenters", "should fail lease whose minimum vcenters cannot be
  satisfied", and "should reject leases with contradictory min-vcenters at
  admission" (`test/leases_test.go`)
