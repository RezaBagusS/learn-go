package metrics

import "github.com/prometheus/client_golang/prometheus"

var (

	// MIDLEWARE LAYER
	HTTPDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler", "method", "status"},
	)

	HTTPRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"handler", "method", "status"},
	)

	// HANDLER LAYER
	CacheRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_requests_total",
			Help: "Total cache requests",
		},
		[]string{"key", "status"}, // status: hit/miss/error
	)

	CacheDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_duration_seconds",
			Help:    "Cache operation duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "key"}, // get/set
	)

	// SERVICE LAYER
	ServiceRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "service_requests_total",
			Help: "Total service layer requests",
		},
		[]string{"service", "operation", "status"}, // status: success/error
	)

	ServiceDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "service_duration_seconds",
			Help:    "Service operation duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "operation"},
	)

	BusinessValidationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_validation_errors_total",
			Help: "Total business validation errors",
		},
		[]string{"service", "operation"},
	)

	// REPO LAYER
	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"repository", "operation"}, // operation: select/insert/update/delete
	)

	DBQueryTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_total",
			Help: "Total database queries",
		},
		[]string{"repository", "operation", "status"}, // status: success/error
	)
)

func Init() {
	prometheus.MustRegister(
		HTTPDuration,
		HTTPRequestTotal,
		CacheRequestsTotal,
		CacheDuration,
		ServiceRequestsTotal,
		ServiceDuration,
		BusinessValidationErrors,
		DBQueryDuration,
		DBQueryTotal,
	)
}
