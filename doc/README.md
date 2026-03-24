# vSphere Capacity Manager — user documentation

The vSphere Capacity Manager is a Kubernetes operator that tracks **capacity** (vCPU, memory, networks) per vSphere failure domain and **fulfills Leases** by choosing a **Pool** and **Network** that satisfy each request.

## Contents

| Document | Audience |
|----------|----------|
| [Concepts](concepts.md) | What Pool, Lease, and Network mean |
| [How it works](how-it-works.md) | Reconciliation flow and diagrams |
| [Scheduling](scheduling.md) | `poolSelector`, taints, tolerations, exclude / noSchedule |
| [Purpose-built networks](networks-purpose-built.md) | Adding a Network CR and wiring it to a Pool |
| [CLI](cli.md) | `oc` / `kubectl` and the optional `oc-vcm` plugin |
| [Pools and networks inventory](inventory-pools-networks.md) | Snapshot of CRs in one environment (refresh manually) |
| [CI / Prow / vsphere-elastic](doc.md) | Job configuration, shared dir files, Vault |

Developer build and test commands remain in the [repository README](../README.md).

## API group

All custom resources use API version `vspherecapacitymanager.splat.io/v1`. They are **namespaced**; examples in this repo often use `vsphere-infra-helpers` — use the namespace where your operator runs.
