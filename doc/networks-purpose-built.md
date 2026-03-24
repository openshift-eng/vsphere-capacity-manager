# Purpose-built networks

Use this when vSphere already has a port group (VLAN, IP plan) and you want the capacity manager to **allocate it** like any other network.

## 1. vSphere and IPAM

- Create or reuse a **distributed port group** (name is what you will put in the CR as `spec.portGroupName`).
- Know **pod**, **datacenter**, VLAN id, gateway, machine CIDR, and any IPv6 fields your installs need.

## 2. Create the `Network` CR

- **`metadata.name`**: unique; convention is often `{portgroup}-{datacenter}-{pod}` with suffixes if you shard one VLAN into multiple logical networks.
- Fill **`spec`** to match your subnet (see examples in the [repository README](../README.md) or [doc.md](doc.md)).
- **Optional — lease matching:** set label **`vsphere-capacity-manager.splat-team.io/network-type`** to one of the values allowed on a Lease’s **`spec.network-type`** (`single-tenant`, `multi-tenant`, `nested-multi-tenant`, `public-ipv6`, …).  
  If the label is **missing**, the operator treats the network as **`single-tenant`**.

## 3. Attach the network to a `Pool`

The controller links pools to networks in `getNetworksForPool` (`pkg/controller/leases.go`):

1. For each path in **`pool.spec.topology.networks`**, take the **basename** (last segment of the path). That string must equal **`network.spec.portGroupName`**.
2. **`network.spec.podName`** must equal **`pool.spec.ibmPoolSpec.pod`**.

Add the full vSphere inventory path of the port group to **`spec.topology.networks`** on the Pool (same style as existing pools in your cluster).

## 4. Verify

```sh
oc get network.vspherecapacitymanager.splat.io -n vsphere-infra-helpers
oc get pool.vspherecapacitymanager.splat.io -n vsphere-infra-helpers -o yaml
```

Create a test **Lease** with the right **`network-type`** and ensure it reaches **Fulfilled** using the new network.

## CI jobs

For Prow jobs using **`vsphere-elastic`**, network type is often driven by **`NETWORK_TYPE`** and related env — see [doc.md](doc.md).
