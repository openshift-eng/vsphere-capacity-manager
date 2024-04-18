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

A lease is requested by a CI job through the creation of a `lease` in the `vsphere-infra-helpers` project. A controller will watch for new `leases`. When a unprocessed `lease` is found, the controller will
attempt to find a pool or pools to fulfill the lease.

```sh
TO-DO: example
```

# Releasing a Lease

When a job is done with a lease, it can simply delete the lease and the controller will return the resources to their respective pools.

```sh
TO-DO: example
```

## CRDs

A CRD which defines a `lease` will be created and applied to the build cluster.  This is the CRD CI jobs will leverage to allocate resources.
A fulfilled lease will contain a reference to it's associated pool in addition to the assigned network parameters.  The pool ID is associable
with the current known vCenter environments.

## Edge Cases

### A job takes too long to complete

In this scenario, Boskos will trigger a shutdown of the job via Prow.  One of the final steps that is executed will
release the leases.  If a lease hangs around for 24 hours, that lease will be reaped.
