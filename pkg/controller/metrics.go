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

	PoolCpusAvailable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_cpus_available",
		Help: "The amount of cpus available in a pool",
	}, []string{"namespace", "pool"})

	PoolNetworksAvailable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pool_networks_available",
		Help: "Number of currently used networks",
	}, []string{"namespace", "pool"})

	LeasesInUse = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "leases_in_use",
		Help: "Number of leases in use",
	}, []string{"namespace", "pool"})
)

func InitMetrics() {
	metrics.Registry.MustRegister(PoolMemoryAvailable, PoolNetworksAvailable, PoolCpusAvailable, LeasesInUse)
}
