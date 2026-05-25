package fema

import "github.com/prometheus/client_golang/prometheus"

var (
	nfhlRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fema_nfhl_requests_total",
			Help: "FEMA NFHL ArcGIS point queries by result status",
		},
		[]string{"status"},
	)
	nfhlErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fema_nfhl_errors_total",
			Help: "FEMA NFHL client errors by reason",
		},
		[]string{"reason"},
	)
	enrichListingsUpdatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "fema_enrich_listings_updated_total",
			Help: "Listings rows updated with FEMA flood attributes",
		},
	)
	enrichBatchDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "fema_enrich_batch_duration_seconds",
			Help:    "Duration of fema.flood_enrich_batch job runs",
			Buckets: prometheus.DefBuckets,
		},
	)
	circuitBreakerOpen = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "fema_circuit_breaker_open",
			Help: "1 when FEMA NFHL circuit breaker is open",
		},
	)
)

func init() {
	prometheus.MustRegister(
		nfhlRequestsTotal,
		nfhlErrorsTotal,
		enrichListingsUpdatedTotal,
		enrichBatchDurationSeconds,
		circuitBreakerOpen,
	)
}
