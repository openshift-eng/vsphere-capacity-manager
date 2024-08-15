# vsphere-capacity-manager

## Overview

A scheduler which aims to distribute OpenShift clusters among a pool of vCenters, datacenters, and clusters.  In an environment with 
number vCenters, datacenter, and clusters, ensuring that an OpenShift cluster is being installed in to a environment with sufficient
capactiy.

![overview](/doc/vSphere%20Resource%20Manager.png)

## Teminology

### Pools

`Pools` are described by a CRD which defines the available capacity for a vSphere failure domain.  A vSphere failure domain describes
a combination of vCenter, datacenter, cluster, and available port groups.  `Pools` 

### Lease

`Leases` are described by a CRD which define the resources that are required from a failure domain. `Leases` are scoped to a
single failure domain.  If multiple failure domains are required, a `lease` for each failure domain must be created.


## Creating a Lease

A Lease is a simple CRD which desribes the requirements of the lease. The number of vcpus, memory, and networks is required. `spec.networks`
is restricted to 1.

```yaml
apiVersion: vspherecapacitymanager.splat.io/v1
kind: Lease
metadata:
  name: sample-lease-0
  namespace: vsphere-infra-helpers
  labels:
    boskos-lease-id: "test-id"
spec:
  requiredPool: <optional: name of the required pool>
  vcpus: 24
  memory: 96
  networks: 1
```

When a `Lease` is fulfilled, `status.phase` will be set to `Fulfilled`.  Additionally, when fulfilled, the `Lease` `

## Defining a Pool

A Pool desribes the resources which are made available for a specific failure domain. The number of vcpus, memory, and networks is required. `spec.topology.networks`
describes the full path of portgroups associated with the pools.

```yaml
piVersion: vspherecapacitymanager.splat.io/v1
kind: Pool
metadata:
  name: vcs8e-vc.ocp2.dev.cluster.com-ibmcloud-vcs-ci-workload
  namespace: vsphere-infra-helpers
spec:
  exclude: true
  ibmPoolSpec:
    datacenter: dalxx
    pod: dalxx.podyy
  memory: 2684
  name: vcs8e-vc.ocp2.dev.cluster.com-IBMCloud-vcs-ci-workload
  noSchedule: false
  region: us-east
  server: vcs8e-vc.ocp2.dev.cluster.com
  storage: 0
  topology:
    computeCluster: /IBMCloud/host/vcs-ci-workload
    datacenter: IBMCloud
    datastore: /IBMCloud/datastore/vsanDatastore
    networks:
    - /IBMCloud/network/ci-vlan-1302
    - /IBMCloud/network/ci-vlan-1300
    - /IBMCloud/network/ci-vlan-1298
    - /IBMCloud/network/ci-vlan-1296
    - /IBMCloud/network/ci-vlan-1289
    - /IBMCloud/network/ci-vlan-1287
    - /IBMCloud/network/ci-vlan-1284
    - /IBMCloud/network/ci-vlan-1279
    - /IBMCloud/network/ci-vlan-1274
    - /IBMCloud/network/ci-vlan-1272
    - /IBMCloud/network/ci-vlan-1271
    - /IBMCloud/network/ci-vlan-1260
    - /IBMCloud/network/ci-vlan-1255
    - /IBMCloud/network/ci-vlan-1254
    - /IBMCloud/network/ci-vlan-1249
    - /IBMCloud/network/ci-vlan-1246
    - /IBMCloud/network/ci-vlan-1243
    - /IBMCloud/network/ci-vlan-1240
    - /IBMCloud/network/ci-vlan-1238
    - /IBMCloud/network/ci-vlan-1237
    - /IBMCloud/network/ci-vlan-1235
    - /IBMCloud/network/ci-vlan-1234
    - /IBMCloud/network/ci-vlan-1233
    - /IBMCloud/network/ci-vlan-1232
    - /IBMCloud/network/ci-vlan-1229
    - /IBMCloud/network/ci-vlan-1227
    - /IBMCloud/network/ci-vlan-1225
    - /IBMCloud/network/ci-vlan-1207
    - /IBMCloud/network/ci-vlan-1197
    - /IBMCloud/network/ci-vlan-1148
    - /IBMCloud/network/ci-vlan-956
  vcpus: 240
```

## Defining a Network

A `Network` defines a VLAN, subnet, and location in a datacenter.

```yaml
apiVersion: vspherecapacitymanager.splat.io/v1
kind: Network
metadata:
  name: ci-vlan-1296-dal10-dal10.pod03
  namespace: vsphere-infra-helpers
spec:
  cidr: 25
  cidrIPv6: 64
  datacenterName: dal10
  gateway: 10.94.169.1
  gatewayipv6: fd65:a1a8:60ad:1296::2
  ipAddressCount: 128
  ipAddresses:
  - 10.94.169.0
  - 10.94.169.1
  - 10.94.169.2
  - 10.94.169.3
  - 10.94.169.4
  - 10.94.169.5
  - 10.94.169.6
  - 10.94.169.7
  - 10.94.169.8
  - 10.94.169.9
  - 10.94.169.10
  - 10.94.169.11
  - 10.94.169.12
  - 10.94.169.13
  - 10.94.169.14
  - 10.94.169.15
  - 10.94.169.16
  - 10.94.169.17
  - 10.94.169.18
  - 10.94.169.19
  ipv6prefix: fd65:a1a8:60ad:1296::/64
  machineNetworkCidr: 10.94.169.0/25
  netmask: 255.255.255.128
  podName: dal10.pod03
  portGroupName: ci-vlan-1296
  primaryRouterHostname: bcr03a.dal10
  startIPv6Address: fd65:a1a8:60ad:1296::4
  subnetType: SECONDARY_ON_VLAN
  vlanId: "1296"
status: {}
```

Networks are associated with `Pools` by the datacenter and pod in which they reside.

## Allocation Strategy

### Pool Configuration

By default, a defined `Pool` will be available for scheduling by any `Lease`. However, `Pools` can be configured to be excluded
from scheduling unless a lease specifically requests it. For pools which exist for specific use cases, this prevents those pools
from being overwhelmed with clusters unrelated to the use case.  

#### Unscheduling a Pool

A pool can be removed from consideration from scheduling by setting `spec.noSchedule` to true. When unscheduled, any leases associated
with the pool will be allowed to remain active.  Newly created leases, however, will not be able to schedule to the pool.

#### Excluding a Pool

A pool can be excluded from consideration unless a lease specifically requests it.  This enables use cases where a pool provides some
unique environment, or configuration, which warrants intentional scheduling to the pool.  To exclude a pool from scheduling, set 
`spec.exclude` to true.

To request a specific pool, a Lease must set `spec.requiredPool` to the name of the pool.

TO-DO: implement a poolSelector paradigm

## Networks

Networks correlate with vSphere port groups.  OpenShift cluster typically must have all nodes on a common subnet, particularly for IPI.  
To accommodate this, the scheduler will attempt to determine if a related lease already has a network lease.  A lease is determined to be related if:

1. The lease has a label `boskos-lease-id`
2. Another lease has the same value for the `boskos-lease-id` label
3. The lease resides in the same vCenter

When leases cross vCenters, those leases will have different network leases.

# `oc` Plugin Installation

An `oc` plugin has been created which enables easier mangagement of the vsphere capacity manager.  To install this plugin:

```sh
cp plugin/oc-vcm /usr/local/bin

oc vcm
usage: oc-vcm [-h]
              {cordon,uncordon,exclude,include,set-capacity,add-vlan,drop-vlan,status,networks,split-network} ...

Process some commands.

positional arguments:
  {cordon,uncordon,exclude,include,set-capacity,add-vlan,drop-vlan,status,networks,split-network}
                        Command to execute
    cordon              Cordon a pool
    uncordon            Uncordon a pool
    exclude             Exclude a pool
    include             Include a pool
    set-capacity        Set capacity
    add-vlan            Add VLAN
    drop-vlan           Drop VLAN
    status              Get status of the capacity manager
    networks            List networks
    split-network       Split a network

options:
  -h, --help            show this help message and exit
```