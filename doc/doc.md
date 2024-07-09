# Overview

vSphere CI requires the use of mutliple environments in order to support the number of jobs and various required configurations. Historically, this has been handled by the creation of purpose targeted lease pools. While this has worked, some environments are overutilized while some environments are idle.  The VCM handles scheduling jobs to the most appropriate environment based on the requirements of the job and the utilization of environments. 

# Job Configuration

Jobs are migrated to VCM by moving the job to the `vsphere-elastic` cluster profile. For most jobs, that is all that needs to be done. Jobs can, however, be run in specific environments. For example, if a job needs to run on a specific vCenter. 

## Job Customization

Four environment variables are available for job configuration.

| Variable              | Default   | Description                    |
|-----------------------|-----------|--------------------------------|
| POOLS                 | *empty*   | space seperated list of pool names required by the job. |
| NETWORK_TYPE          | single-tenant | the type of network required by the job. If not defined, the job will be assigned a network type based on the JOB_SAFE_NAME. |
| OPENSHIFT_REQUIRED_CORES | 24 | the number of vCPUs assigned to the job. |
| OPENSHIFT_REQUIRED_MEMORY | 96 | the amount of memory assigned to the job. |

### Getting Available Pools

Available pools can be retrieved by running `oc get pools.vspherecapacitymanager.splat.io -n vsphere-infra-helpers`. 

```sh
$ oc get pools.vspherecapacitymanager.splat.io -n vsphere-infra-helpers
NAME                                                                  VCPUS   MEMORY(GB)   NETWORKS   DEGRADED   DISABLED   EXCLUDED
vcenter-7-nested-dal10.pod03                                          96      384          3                     false      true
vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-1   112     447          2                     false      false
vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-2   112     447          2                     false      false
vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-cicluster       224     2944         2                     false      false
```

Pools have a defined capacity and associated portgroups. The scheduler chooses a pool based on its utilization, job configuration, and if a given pool is `disabled` or `excluded`.

If a pool is `disabled`, it is equivalent to a `node` being cordoned. Jobs running on a `disabled` pool will not be evicted, but new jobs will not be scheduled. If a pool is `excluded`, it will not be chosen unless a job specifically requests it.

#### Specifying a Pool in Lease

In the below example, a job is configured to use the `vcenter-7-nested-dal10.pod03` pool.

```yaml
- as: e2e-vsphere-ovn
  interval: 168h
  steps:
    cluster_profile: vsphere-elastic
    env:
      POOLS: "vcenter-7-nested-dal10.pod03"
      TEST_SKIPS: provisioning should mount multiple PV pointing to the same storage
        on the same node
    observers:
      enable:
      - observers-resource-watch
    workflow: openshift-e2e-vsphere-ovn
```

A job can request mutliple pools. This is useful for testing multiple failure domains or multiple vCenters.

```yaml
- as: e2e-vsphere-ovn
  interval: 168h
  steps:
    cluster_profile: vsphere-elastic
    env:
      POOLS: "vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-cicluster vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-1 vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-2"
      TEST_SKIPS: provisioning should mount multiple PV pointing to the same storage
        on the same node
    observers:
      enable:
      - observers-resource-watch
    workflow: openshift-e2e-vsphere-ovn
```

#### Resource Allocation

By default, a job is assigned 24 vCPUs and 96GB of RAM. If a job split among multiple pools, each pool is assigned resources using the following equations:

- POOL_CPUS_USED = 24 / NUMBER_OF_POOLS
- POOL_MEMORY_USED = 96 / NUMBER_OF_POOLS

__WARNING: Resources will eventually restricted by resource pool based on assigned resources. If you require more than the default allocation, please be sure to update the resource requirements of your job. Jobs which signficantly exceed the resource allocation may be disabled or restricted to the allocated resource values.__

# Job Development

## Configuration Files

The step `ipi-conf-vsphere-check-vcm` is responsible for deriving configuration used by downstream steps. This configuration is kept in `${SHARED_DIR}`. Deriving configuration _should not_ be performed in other steps. This ensures consistent configuration is made available to all jobs.

| File              |  Description                    |
|-------------------|------------------------------------|
| platform.json | platform spec in JSON. This is used to build variables for UPI jobs. |
| platform.yaml | platform spec in YAML. This is used to derive install-config.yaml. |
| dvs.json | JSON map of networks and associated distributed switch UUIDs. 
| vips.txt | List of VIPs. Each line contains one VIP.
| govc.sh  | environment variables associated with a single lease. This is useful for jobs which only use a single lease.
| vsphere_context.sh | identical to govc.sh and is kept for compatibility purposes.
| subnets.json | JSON map of each network resource allocated to the job.
| LEASE_*.json | each lease associated with the job is stored as JSON. LEASE_single.json can be used for single lease jobs.
| POOL_*.json | each pool associated with the job is stored as JSON. POOL_single.json can be used for single lease jobs.
| NETWORK_*.json | each network associated with the job is stored as JSON. NETWORK_single.json can be used for single lease jobs.

__WARNING: Please be careful when attempting to log the content of these files. Some of these files may contain credentials. Although these environments are not accessible it is still important to use caution.__

## VCM Steps

VCM is implemented in such a way that jobs which are not migrated to the `vsphere-elastic` cluster profile continue to operate as they always have. To facilitate this, some steps have parallel VCM specific steps.

| Original Step | VCM Step
|---------------|----
| ipi-conf-vsphere-check | ipi-conf-vsphere-check-vcm 
| ipi-conf-vsphere-vips | ipi-conf-vsphere-vips-vcm
| ipi-deprovision-vsphere-diags | ipi-deprovision-vsphere-diags-vcm
| upi-conf-vsphere-ova | upi-conf-vsphere-ova-vcm
| upi-conf-vsphere-zones | upi-conf-vsphere-zones-vcm

For each of these steps, a check is performed very early in the script. 

For VCM steps:
```sh
if [[ "${CLUSTER_PROFILE_NAME:-}" != "vsphere-elastic" ]]; then
  echo "using legacy sibling of this step"
  exit 0
fi
```

For equivilent non-VCM steps:
```sh
if [[ "${CLUSTER_PROFILE_NAME:-}" == "vsphere-elastic" ]]; then
  echo "using VCM sibling of this step"
  exit 0
fi
```

This allows a single chain to support both VCM and legacy jobs.

# Appendix

## Networks

__NOTE: Generally, jobs won't need to introspect networks since `platform.json`, `platform.yaml`, and `vips.txt` already condense the content of networks(s) in to a format that is required for a successful installation.__

Networks are vSphere portgroups which are associated with `pools`. Each pool has a slice of associated portgroups. Each portgroup is trunked to a VLAN. Traditionally, jobs are assigned a portgroup which is exclusive to a VLAN. OpenShift clusters can share a VLAN with other clusters. Eventually, most jobs will be transistion to a shared VLAN. 

Networks can be retrieved by running:

```sh
$ oc get networks.vspherecapacitymanager.splat.io -n vsphere-infra-helpers
NAME                                PORT GROUP                      POD
ci-vlan-1108-1-dal10-dal10.pod03    ci-vlan-1108                    dal10.pod03
ci-vlan-1148-dal10-dal10.pod03      ci-vlan-1148                    dal10.pod03
ci-vlan-1161-dal10-dal10.pod03      ci-vlan-1161                    dal10.pod03
ci-vlan-1190-dal10-dal10.pod03      ci-vlan-1190                    dal10.pod03
```

Each `network` resource contains information about the network configuration and datacenter details.

```yaml
apiVersion: vspherecapacitymanager.splat.io/v1
kind: Network
metadata:
  creationTimestamp: "2024-05-22T13:32:56Z"
  finalizers:
  - vsphere-capacity-manager.splat-team.io/network-finalizer
  generation: 2
  name: ci-vlan-958-1-dal10-dal10.pod03
  namespace: vsphere-infra-helpers
  resourceVersion: "249420087"
  uid: a8427859-6cc5-4869-a8d5-44e6286f608f
spec:
  cidr: 25
  cidrIPv6: 64
  datacenterName: dal10
  gateway: 10.93.152.1
  gatewayipv6: fd65:a1a8:60ad:958::2
  ipAddressCount: 128
  ipAddresses:
  - 10.93.152.0
  - 10.93.152.1
  - 10.93.152.2
  - 10.93.152.3
  - 10.93.152.4
  - 10.93.152.5
  - 10.93.152.6
  - 10.93.152.7
  - 10.93.152.8
  - 10.93.152.9
  - 10.93.152.10
  - 10.93.152.11
  - 10.93.152.12
  - 10.93.152.13
  - 10.93.152.14
  - 10.93.152.15
  - 10.93.152.16
  - 10.93.152.17
  - 10.93.152.18
  - 10.93.152.19
  ipv6prefix: fd65:a1a8:60ad:958::/64
  machineNetworkCidr: 10.93.152.0/25
  netmask: 255.255.255.128
  podName: dal10.pod03
  portGroupName: ci-vlan-958
  primaryRouterHostname: bcr03a.dal10
  startIPv6Address: fd65:a1a8:60ad:958::4
  subnetType: SECONDARY_ON_VLAN
  vlanId: "958"
status: {}
```

See https://github.com/openshift-splat-team/vsphere-capacity-manager/blob/main/pkg/apis/vspherecapacitymanager.splat.io/v1/network_types.go for a  comprehensive description of fields, labels, and annotations.

## Leases

__NOTE: Generally, jobs won't need to introspect leases since `platform.json` and `platform.yaml` already condense the content of lease(s) in to a format that is required for a successful installation.__

Leases can be retrieved(if you have access) by running `oc get leases.vspherecapacitymanager.splat.io -n vsphere-infra-helpers`

```sh
$ oc get leases.vspherecapacitymanager.splat.io -n vsphere-infra-helpers
NAME                       VCPUS   MEMORY(GB)   POOL                                                                  NETWORK                            PHASE
vsphere-elastic-0-9wf5v    24      96           vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-cicluster       ci-vlan-1272-dal10-dal10.pod03     Fulfilled
vsphere-elastic-13-m4xh7   24      96           vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-cicluster       ci-vlan-1279-dal10-dal10.pod03     Fulfilled
vsphere-elastic-18-4l6p2   24      96           vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-2   ci-vlan-938-1-dal10-dal10.pod03    Fulfilled
vsphere-elastic-20-f7dpr   24      96           vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-cicluster       ci-vlan-847-1-dal10-dal10.pod03    Fulfilled
```

See https://github.com/openshift-splat-team/vsphere-capacity-manager/blob/main/pkg/apis/vspherecapacitymanager.splat.io/v1/lease_types.go for a  comprehensive description of fields, labels, and annotations.

## Pools 

__NOTE: Generally, jobs won't need to introspect pools since `platform.json` and `platform.yaml` already condense the content of pool(s) in to a format that is required for a successful installation.__

Available pools can be retrieved by running `oc get pools.vspherecapacitymanager.splat.io -n vsphere-infra-helpers`. 

```sh
$ oc get pools.vspherecapacitymanager.splat.io -n vsphere-infra-helpers
NAME                                                                  VCPUS   MEMORY(GB)   NETWORKS   DEGRADED   DISABLED   EXCLUDED
vcenter-7-nested-dal10.pod03                                          96      384          3                     false      true
vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-1   112     447          2                     false      false
vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-1-cicluster-2   112     447          2                     false      false
vcenter.ci.ibmc.devcluster.openshift.com-cidatacenter-cicluster       224     2944         2                     false      false
```

### Pool Authentication

Each pool is annotated with `ci-auth-path` which is the path to the Vault mounted credentials. 

```yaml
apiVersion: vspherecapacitymanager.splat.io/v1
kind: Pool
metadata:
  annotations:
    ci-auth-path: /var/run/vault/vsphere-ibmcloud-ci/secrets-vcenter-7.sh
```

Be sure that your step mounts the credentials with:

```yaml
  credentials:
  ...
  - namespace: test-credentials
    name: vsphere-ibmcloud-config
    mount_path: /var/run/vault/vsphere-ibmcloud-config
  ...
```