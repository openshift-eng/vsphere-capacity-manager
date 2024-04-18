# vsphere-capacity-manager

## Overview

A scheduling layer on top of Boskos which aims to spread the load of jobs evenly among a pool
of vCenters.

![overview](/doc/vSphere%20Resource%20Manager.png)

## Allocation Strategy

Be default, leases are assigned to the least utilized vCenter.

## Deployment

This service will be deployed on a managing OpenShift cluster.  Prow pods running on the managing cluster will access
this service as a Kubernetes service.

## Storing State

State will be stored in configmap adjacent to the deployment.

## Acquiring a Lease

To acquire a lease(or leases), the desired resources are provided to the `acquire` API call.  A resource defines the number of required vCPUs, memory, and storage.  Additionally, vCenters specifies how many pools are needed.  The resource specification will apply to each pool.  If more than one vCenter is specified, the quantity of vCPUs, memory, and storage will be required for each vCenter.

```sh
curl --location --request POST 'http://localhost:8080/acquire' \
--header 'Content-Type: application/json' \
--data-raw '{
    "vcpus": 24,
    "memory": 96,
    "storage": 700,
    "vcenters": 1
}'
```

If the desired leases are acquired, the lease details are return.  While not shown below, the lease will also include the port group
and details from subnet.json.

```json
[
  {
    "spec": {
      "vcpus": 24,
      "memory": 96,
      "storage": 700,
      "vcenters": 0
    },
    "status": {
      "resources": [
        {
          "spec": {
            "vcpus": 24,
            "memory": 96,
            "storage": 700,
            "vcenters": 1
          },
          "status": {
            "vcpus": 0,
            "memory": 0,
            "storage": 0,
            "pool": "",
            "lease": ""
          },
          "metadata": {
            "creationTimestamp": null
          },
          "type": {}
        }
      ],
      "leased-at": "2024-04-17 15:32:06.9265657 -0400 EDT m=+201.560691053",
      "boskos-lease-id": "",
      "pool": "pool1"
    },
    "metadata": {
      "creationTimestamp": null
    },
    "type": {}
  }
]
```

### Getting Pools

The allocation level of each pool can be retrieved by a call to show-pools.  The status will show
any leases associated with the pool as well as the percentage of exhaustion for resources in the
pool.

Note: To reduce complexity, the individual vCenters are not being contacted to get real-time utilization
metrics.  This could be done but I fear it would make this more brittle with little benefit.

```sh
$ curl localhost:8080/show-pools
```

```json
[
  {
    "spec": {
      "vcpus": 120,
      "memory": 1600,
      "storage": 10000,
      "vcenters": 0,
      "name": "pool1",
      "vcenter": "vcenter1",
      "datacenter": "datacenter1",
      "cluster": "cluster1",
      "datastore": "datastore1",
      "networks": null
    },
    "status": {
      "vcpus-usage": 0.4,
      "memory-usage": 0.12,
      "datastore-usage": 0.14,
      "leases": [
        {
          "spec": {
            "vcpus": 24,
            "memory": 96,
            "storage": 700,
            "vcenters": 0
          },
          "status": {
            "resources": [
              {
                "spec": {
                  "vcpus": 24,
                  "memory": 96,
                  "storage": 700,
                  "vcenters": 1
                },
                "status": {
                  "vcpus": 0,
                  "memory": 0,
                  "storage": 0,
                  "pool": "",
                  "lease": ""
                },
                "metadata": {
                  "creationTimestamp": null
                },
                "type": {}
              }
            ],
            "leased-at": "2024-04-17 15:28:48.135124729 -0400 EDT m=+2.769250092",
            "boskos-lease-id": "",
            "pool": "pool1"
          },
          "metadata": {
            "creationTimestamp": null
          },
          "type": {}
        },
        {
          "spec": {
            "vcpus": 24,
            "memory": 96,
            "storage": 700,
            "vcenters": 0
          },
          "status": {
            "resources": [
              {
                "spec": {
                  "vcpus": 24,
                  "memory": 96,
                  "storage": 700,
                  "vcenters": 1
                },
                "status": {
                  "vcpus": 0,
                  "memory": 0,
                  "storage": 0,
                  "pool": "",
                  "lease": ""
                },
                "metadata": {
                  "creationTimestamp": null
                },
                "type": {}
              }
            ],
            "leased-at": "2024-04-17 15:32:06.9265657 -0400 EDT m=+201.560691053",
            "boskos-lease-id": "",
            "pool": "pool1"
          },
          "metadata": {
            "creationTimestamp": null
          },
          "type": {}
        }
      ],
      "network-usage": 0
    }
  },
  {
    "spec": {
      "vcpus": 60,
      "memory": 800,
      "storage": 5000,
      "vcenters": 0,
      "name": "pool2",
      "vcenter": "vcenter2",
      "datacenter": "datacenter2",
      "cluster": "cluster2",
      "datastore": "datastore2",
      "networks": null
    },
    "status": {
      "vcpus-usage": 0.4,
      "memory-usage": 0.12,
      "datastore-usage": 0.14,
      "leases": [
        {
          "spec": {
            "vcpus": 24,
            "memory": 96,
            "storage": 700,
            "vcenters": 0
          },
          "status": {
            "resources": [
              {
                "spec": {
                  "vcpus": 24,
                  "memory": 96,
                  "storage": 700,
                  "vcenters": 1
                },
                "status": {
                  "vcpus": 0,
                  "memory": 0,
                  "storage": 0,
                  "pool": "",
                  "lease": ""
                },
                "metadata": {
                  "creationTimestamp": null
                },
                "type": {}
              }
            ],
            "leased-at": "2024-04-17 15:29:21.919057428 -0400 EDT m=+36.553182791",
            "boskos-lease-id": "",
            "pool": "pool2"
          },
          "metadata": {
            "creationTimestamp": null
          },
          "type": {}
        }
      ],
      "network-usage": 0
    }
  },
  {
    "spec": {
      "vcpus": 40,
      "memory": 600,
      "storage": 1000,
      "vcenters": 0,
      "name": "pool3",
      "vcenter": "vcenter3",
      "datacenter": "datacenter3",
      "cluster": "cluster3",
      "datastore": "datastore3",
      "networks": null
    },
    "status": {
      "vcpus-usage": 0.6,
      "memory-usage": 0.16,
      "datastore-usage": 0.7,
      "leases": [
        {
          "spec": {
            "vcpus": 24,
            "memory": 96,
            "storage": 700,
            "vcenters": 0
          },
          "status": {
            "resources": [
              {
                "spec": {
                  "vcpus": 24,
                  "memory": 96,
                  "storage": 700,
                  "vcenters": 1
                },
                "status": {
                  "vcpus": 0,
                  "memory": 0,
                  "storage": 0,
                  "pool": "",
                  "lease": ""
                },
                "metadata": {
                  "creationTimestamp": null
                },
                "type": {}
              }
            ],
            "leased-at": "2024-04-17 15:32:02.632067175 -0400 EDT m=+197.266192538",
            "boskos-lease-id": "",
            "pool": "pool3"
          },
          "metadata": {
            "creationTimestamp": null
          },
          "type": {}
        }
      ],
      "network-usage": 0
    }
  }
]
```
## Edge Cases

### A job takes too long to complete

In this scenario, Boskos will trigger a shutdown of the job via Prow.  One of the final steps that is executed will
release the leases.  If a lease hangs around for 24 hours, that lease will be reaped.
