package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "opsowl",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests by method, route, and status code.",
	}, []string{"method", "route", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "opsowl",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "route"})

	httpInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "opsowl",
		Name:      "http_requests_in_flight",
		Help:      "Current number of in-flight HTTP requests.",
	})
)

// Metrics records Prometheus metrics for every HTTP request.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || r.URL.Path == "/health" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		metricsRec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		httpInFlight.Inc()
		start := time.Now()

		next.ServeHTTP(metricsRec, r)

		httpInFlight.Dec()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(metricsRec.status)

		// Use a bucketed route label to avoid high cardinality from path params
		route := routeLabel(r)
		httpRequests.WithLabelValues(r.Method, route, status).Inc()
		httpDuration.WithLabelValues(r.Method, route).Observe(duration)
	})
}

// routeLabel extracts a low-cardinality route label from the request.
// Chi's RouteContext would give us the pattern, but this middleware runs
// before routing resolves, so we collapse path segments after /api/v1/.
func routeLabel(r *http.Request) string {
	path := r.URL.Path
	// Collapse UUIDs and numeric IDs to keep cardinality manageable
	result := make([]byte, 0, len(path))
	segment := make([]byte, 0, 36)
	inSegment := false

	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' {
			if inSegment {
				if looksLikeID(segment) {
					result = append(result, []byte("{id}")...)
				} else {
					result = append(result, segment...)
				}
				segment = segment[:0]
			}
			result = append(result, c)
			inSegment = false
		} else {
			if !inSegment {
				inSegment = true
			}
			segment = append(segment, c)
		}
	}
	if len(segment) > 0 {
		if looksLikeID(segment) {
			result = append(result, []byte("{id}")...)
		} else {
			result = append(result, segment...)
		}
	}
	return string(result)
}

func looksLikeID(s []byte) bool {
	// UUID: 36 chars with hyphens
	if len(s) == 36 && s[8] == '-' && s[13] == '-' {
		return true
	}
	// Pure numeric
	if len(s) > 0 {
		allDigits := true
		for _, c := range s {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	return false
}
