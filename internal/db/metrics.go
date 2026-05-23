package db

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	replicaLatencyMs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "replica_latency_ms",
			Help: "Last measured round-trip latency in milliseconds for each PostgreSQL read replica.",
		},
		[]string{"host"},
	)

	replicaSelectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "replica_selected_total",
			Help: "Number of times each replica host was selected by GetBestReplica.",
		},
		[]string{"host"},
	)

	replicaDiscoveryErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "replica_discovery_errors_total",
			Help: "Total Patroni cluster discovery or probe cycle errors.",
		},
	)
)

func init() {
	prometheus.MustRegister(replicaLatencyMs, replicaSelectedTotal, replicaDiscoveryErrorsTotal)
}

func setReplicaLatency(host string, ms float64) {
	replicaLatencyMs.WithLabelValues(host).Set(ms)
}

func deleteReplicaLatency(host string) {
	replicaLatencyMs.DeleteLabelValues(host)
}

func incReplicaSelected(host string) {
	replicaSelectedTotal.WithLabelValues(host).Inc()
}

func incDiscoveryErrors() {
	replicaDiscoveryErrorsTotal.Inc()
}
