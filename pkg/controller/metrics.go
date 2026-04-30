package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	PoolMemoryAvailable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_memory_available",
		Help: "The amount of memory available in a pool",
	}, []string{"namespace", "pool"})

	PoolMemoryTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_memory_total",
		Help: "The total amount of memory of a pool",
	}, []string{"namespace", "pool"})

	PoolCpusAvailable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_cpus_available",
		Help: "The amount of cpus available in a pool",
	}, []string{"namespace", "pool"})

	PoolCpusTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_cpus_total",
		Help: "The total amount of cpus of a pool",
	}, []string{"namespace", "pool"})

	PoolNetworksAvailable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_networks_available",
		Help: "Number of available (not in use) networks per pool",
	}, []string{"namespace", "pool"})

	PoolNetworksTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_networks_total",
		Help: "Total number of networks in a pool",
	}, []string{"namespace", "pool"})

	PoolNetworksAvailableByType = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_networks_available_by_type",
		Help: "Number of available (not in use) networks per pool, broken down by network type",
	}, []string{"namespace", "pool", "networkType"})

	PoolNetworksTotalByType = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_networks_total_by_type",
		Help: "Total number of networks per pool, broken down by network type",
	}, []string{"namespace", "pool", "networkType"})

	PoolVcpusUtilizationRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_vcpus_utilization_ratio",
		Help: "Ratio of vCPUs in use to total available (with overcommit) per pool",
	}, []string{"namespace", "pool"})

	PoolMemoryUtilizationRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_memory_utilization_ratio",
		Help: "Ratio of memory in use to total available per pool",
	}, []string{"namespace", "pool"})

	PoolNetworksUtilizationRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_networks_utilization_ratio",
		Help: "Ratio of networks in use to total available per pool",
	}, []string{"namespace", "pool"})

	PoolNoSchedule = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_no_schedule",
		Help: "Whether pool has scheduling disabled (1=disabled, 0=enabled)",
	}, []string{"namespace", "pool"})

	PoolExcluded = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_excluded",
		Help: "Whether pool is excluded from default scheduling (1=excluded, 0=included)",
	}, []string{"namespace", "pool"})

	LeasesInUse = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "leases_in_use",
		Help: "Number of leases in use",
	}, []string{"namespace", "pool"})

	LeaseCounts = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "leases_counts",
		Help: "Counts of active leases",
	}, []string{"namespace", "networkType", "phase"})

	LeaseAgeSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lease_age_seconds",
		Help: "Age of active leases in seconds since creation",
	}, []string{"namespace", "lease", "pool", "networkType"})

	LeaseTransitionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lease_transitions_total",
		Help: "Total number of lease phase transitions",
	}, []string{"namespace", "networkType", "phase"})

	LeaseDelaysTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lease_delays_total",
		Help: "Total number of times leases have been delayed",
	}, []string{"namespace", "networkType"})

	NetworkLeaseCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "network_lease_count",
		Help: "Number of leases currently using each network",
	}, []string{"namespace", "network", "networkType", "pool"})
)

func InitMetrics() {
	metrics.Registry.MustRegister(
		PoolMemoryAvailable, PoolMemoryTotal,
		PoolNetworksAvailable, PoolNetworksTotal,
		PoolNetworksAvailableByType, PoolNetworksTotalByType,
		PoolCpusAvailable, PoolCpusTotal,
		PoolVcpusUtilizationRatio, PoolMemoryUtilizationRatio, PoolNetworksUtilizationRatio,
		PoolNoSchedule, PoolExcluded,
		LeasesInUse, LeaseCounts,
		LeaseAgeSeconds, LeaseTransitionsTotal, LeaseDelaysTotal,
		NetworkLeaseCount,
	)
}
