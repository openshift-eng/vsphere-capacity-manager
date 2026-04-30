# Prometheus Queries

Useful PromQL queries for monitoring the vSphere Capacity Manager. All metrics are exposed on the `/metrics` endpoint.

## Pool Capacity

### CPU utilization per pool

```promql
pool_vcpus_utilization_ratio
```

Pools approaching 1.0 are running out of CPU capacity. Factor in `pool_no_schedule` to exclude disabled pools:

```promql
pool_vcpus_utilization_ratio and on(namespace, pool) pool_no_schedule == 0
```

### Memory utilization per pool

```promql
pool_memory_utilization_ratio
```

### Network utilization per pool

```promql
pool_networks_utilization_ratio
```

### Pools with high utilization (any resource above 80%)

```promql
pool_vcpus_utilization_ratio > 0.8
  or pool_memory_utilization_ratio > 0.8
  or pool_networks_utilization_ratio > 0.8
```

### Available resources per pool

```promql
pool_cpus_available
pool_memory_available
pool_networks_available
```

### Total vs available side by side

```promql
pool_cpus_total - pool_cpus_available
```

## Pool Scheduling State

### Pools excluded from scheduling

```promql
pool_excluded == 1
```

### Pools with scheduling disabled

```promql
pool_no_schedule == 1
```

### Count of schedulable pools

```promql
count(pool_no_schedule == 0 and pool_excluded == 0)
```

## Networks by Type

### Total networks per pool by type

```promql
pool_networks_total_by_type
```

### Available networks per pool by type

```promql
pool_networks_available_by_type
```

### Networks in use per type

```promql
pool_networks_total_by_type - pool_networks_available_by_type
```

### Multi-tenant network availability across all pools

```promql
sum(pool_networks_available_by_type{networkType="multi-tenant"})
```

### Network type breakdown for a specific pool

```promql
pool_networks_total_by_type{pool="devqe-pool-1"}
```

## Network Sharing

### Lease count per network

```promql
network_lease_count
```

### Multi-tenant networks with the most leases

```promql
topk(10, network_lease_count{networkType="multi-tenant"})
```

### Networks with no active leases

```promql
network_lease_count == 0
```

## Leases

### Active lease counts by phase

```promql
leases_counts
```

### Leases currently in Pending or Partial state

```promql
leases_counts{phase=~"Pending|Partial"}
```

### Total leases in use per pool

```promql
leases_in_use
```

## Lease Age

### All lease ages

```promql
lease_age_seconds
```

### Leases older than 7 days

```promql
lease_age_seconds > 86400 * 7
```

### Leases older than 30 days

```promql
lease_age_seconds > 86400 * 30
```

### Top 10 oldest leases

```promql
topk(10, lease_age_seconds)
```

### Average lease age by network type

```promql
avg by (networkType) (lease_age_seconds)
```

## Lease Lifecycle

### Lease transition rate (per 5 minutes)

```promql
rate(lease_transitions_total[5m])
```

### Lease delay rate (per 5 minutes)

```promql
rate(lease_delays_total[5m])
```

### Total leases fulfilled in the last hour

```promql
increase(lease_transitions_total{phase="Fulfilled"}[1h])
```

### Delay-to-fulfillment ratio (high values indicate scheduling pressure)

```promql
rate(lease_delays_total[1h]) / rate(lease_transitions_total{phase="Fulfilled"}[1h])
```

## Alerting Examples

### Alert: pool CPU above 90%

```promql
pool_vcpus_utilization_ratio > 0.9
  and on(namespace, pool) pool_no_schedule == 0
  and on(namespace, pool) pool_excluded == 0
```

### Alert: no multi-tenant networks available

```promql
sum(pool_networks_available_by_type{networkType="multi-tenant"}) == 0
```

### Alert: lease stuck (not fulfilled after 30 minutes)

```promql
lease_age_seconds{networkType=~".+"} > 1800
  and on(namespace, lease) leases_counts{phase="Pending"} > 0
```

### Alert: high delay rate

```promql
rate(lease_delays_total[15m]) > 0.5
```

### Alert: stale lease (older than 14 days)

```promql
lease_age_seconds > 86400 * 14
```
