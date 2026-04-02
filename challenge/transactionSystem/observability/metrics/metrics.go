package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
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
)

func Init() {
	prometheus.MustRegister(HTTPDuration)
	prometheus.MustRegister(HTTPRequestTotal)

	prometheus.MustRegister(CacheRequestsTotal)
	prometheus.MustRegister(CacheDuration)
}
