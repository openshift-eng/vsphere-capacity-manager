# CLI reference

Replace the namespace if yours differs from `vsphere-infra-helpers`.

## List and inspect CRs

```sh
NS=vsphere-infra-helpers

oc get pools.vspherecapacitymanager.splat.io -n "$NS" -o wide
oc get leases.vspherecapacitymanager.splat.io -n "$NS" -o wide
oc get networks.vspherecapacitymanager.splat.io -n "$NS"
```

Describe a single object:

```sh
oc describe pool.vspherecapacitymanager.splat.io/<name> -n "$NS"
```

## Optional `oc-vcm` plugin

The repo ships a helper script — see [repository README](../README.md#oc-plugin-installation). After installing:

```sh
oc vcm
```

Subcommands include `status`, `networks`, pool cordon/uncordon, exclude/include, VLAN helpers, etc.

## Inventory snapshot

To regenerate the tables in [inventory-pools-networks.md](inventory-pools-networks.md), use the refresh section at the bottom of that file.
