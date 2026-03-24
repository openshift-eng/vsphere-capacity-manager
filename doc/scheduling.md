# Scheduling: poolSelector, taints, and tolerations

This page describes how a **Lease** is matched to **Pool** instances beyond raw capacity. Detailed logic lives in `pkg/utils/pools.go` (`GetFittingPools`, `poolMatchesSelector`, `leaseToleratesPoolTaints`).

## poolSelector (on the Lease)

Field: **`spec.poolSelector`** (map of string → string).

Semantics match **Kubernetes `nodeSelector`**: **every** key in the map must exist on the Pool’s **`metadata.labels`** with the **exact same value**. There is no `In` / `NotIn` / set-based selector.

- Empty or omitted → no label constraint.
- Example: only pools labeled `region=us-east`:

```yaml
spec:
  poolSelector:
    region: us-east
```

Ensure the Pool objects carry those labels; otherwise the lease will not schedule.

## Taints and tolerations

**Pools** may define **`spec.taints`** (key, optional value, effect `NoSchedule` or `PreferNoSchedule`).

**Leases** may define **`spec.tolerations`** (operator `Equal` or `Exists`, optional effect).

Rules:

- If a pool has **no** taints, any lease may use it (subject to other rules).
- If a pool has taints, the lease must **tolerate every taint** on that pool. One missing toleration disqualifies the pool.
- **`Exists`** with a key can match that taint key regardless of value; empty key with `Exists` is a broad match (see unit tests in `pkg/utils/pools_test.go`).

Example pool taint:

```yaml
spec:
  taints:
    - key: workload
      value: gpu
      effect: NoSchedule
```

Matching lease:

```yaml
spec:
  tolerations:
    - key: workload
      operator: Equal
      value: gpu
      effect: NoSchedule
```

## Interaction with other pool controls

| Mechanism | Effect |
|-----------|--------|
| **`spec.exclude`** on Pool | Pool is skipped **unless** the lease names it with **`spec.required-pool`** (exact Pool metadata name). |
| **`spec.noSchedule`** on Pool | Pool cannot take **new** leases; existing ones remain. |
| **`spec.required-pool`** on Lease | Lease may **only** use that pool name if it passes capacity and taint/selector checks. |
| **`poolSelector`** | Pool must match **all** listed labels. |
| **Taints / tolerations** | Every pool taint must be tolerated. |

Capacity (vCPU, memory, networks), excluded pools, and network availability are still evaluated after these gates.

## Network type

Independent of pool selection, the lease’s **`spec.network-type`** (e.g. `single-tenant`, `multi-tenant`) filters which **Network** CRs are eligible; see [Purpose-built networks](networks-purpose-built.md).
