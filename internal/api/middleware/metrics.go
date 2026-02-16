package middleware

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects HTTP request metrics in a Prometheus-compatible format.
type Metrics struct {
	requestsTotal   sync.Map // key: "method:status" -> *int64
	requestDuration sync.Map // key: "method:path" -> *durationBuckets
	activeRequests  int64
}

type durationBuckets struct {
	mu    sync.Mutex
	sum   float64
	count int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			atomic.AddInt64(&m.activeRequests, 1)

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			atomic.AddInt64(&m.activeRequests, -1)
			duration := time.Since(start).Seconds()

			// Count requests by method+status
			key := fmt.Sprintf("%s:%d", r.Method, rw.status)
			counter, _ := m.requestsTotal.LoadOrStore(key, new(int64))
			atomic.AddInt64(counter.(*int64), 1)

			// Track duration by method+path pattern
			pathKey := fmt.Sprintf("%s:%s", r.Method, normalizeMetricsPath(r.URL.Path))
			buckets, _ := m.requestDuration.LoadOrStore(pathKey, &durationBuckets{})
			db := buckets.(*durationBuckets)
			db.mu.Lock()
			db.sum += duration
			db.count++
			db.mu.Unlock()
		})
	}
}

// Handler serves the /metrics endpoint in Prometheus text exposition format.
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// Active requests gauge
		fmt.Fprintf(w, "# HELP harbor_http_active_requests Number of active HTTP requests.\n")
		fmt.Fprintf(w, "# TYPE harbor_http_active_requests gauge\n")
		fmt.Fprintf(w, "harbor_http_active_requests %d\n\n", atomic.LoadInt64(&m.activeRequests))

		// Request totals
		fmt.Fprintf(w, "# HELP harbor_http_requests_total Total number of HTTP requests.\n")
		fmt.Fprintf(w, "# TYPE harbor_http_requests_total counter\n")

		var totalKeys []string
		m.requestsTotal.Range(func(key, _ interface{}) bool {
			totalKeys = append(totalKeys, key.(string))
			return true
		})
		sort.Strings(totalKeys)
		for _, key := range totalKeys {
			val, _ := m.requestsTotal.Load(key)
			method, status := splitMetricsKey(key)
			fmt.Fprintf(w, "harbor_http_requests_total{method=%q,status=%q} %d\n",
				method, status, atomic.LoadInt64(val.(*int64)))
		}

		// Request duration
		fmt.Fprintf(w, "\n# HELP harbor_http_request_duration_seconds HTTP request duration in seconds.\n")
		fmt.Fprintf(w, "# TYPE harbor_http_request_duration_seconds summary\n")

		var durationKeys []string
		m.requestDuration.Range(func(key, _ interface{}) bool {
			durationKeys = append(durationKeys, key.(string))
			return true
		})
		sort.Strings(durationKeys)
		for _, key := range durationKeys {
			val, _ := m.requestDuration.Load(key)
			db := val.(*durationBuckets)
			db.mu.Lock()
			sum := db.sum
			count := db.count
			db.mu.Unlock()
			method, path := splitMetricsKey(key)
			fmt.Fprintf(w, "harbor_http_request_duration_seconds_sum{method=%q,path=%q} %.6f\n", method, path, sum)
			fmt.Fprintf(w, "harbor_http_request_duration_seconds_count{method=%q,path=%q} %d\n", method, path, count)
		}
	}
}

func splitMetricsKey(key string) (string, string) {
	for i, c := range key {
		if c == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// normalizeMetricsPath replaces UUIDs and numeric IDs with {id} to group metrics.
func normalizeMetricsPath(path string) string {
	parts := make([]byte, 0, len(path))
	i := 0
	for i < len(path) {
		if path[i] == '/' {
			parts = append(parts, '/')
			i++
			// Check if the next segment looks like a UUID or numeric ID
			j := i
			for j < len(path) && path[j] != '/' {
				j++
			}
			segment := path[i:j]
			if isIDSegment(segment) {
				parts = append(parts, "{id}"...)
			} else {
				parts = append(parts, segment...)
			}
			i = j
		} else {
			parts = append(parts, path[i])
			i++
		}
	}
	return string(parts)
}

func isIDSegment(s string) bool {
	if len(s) == 0 {
		return false
	}
	// UUID pattern: 8-4-4-4-12 hex chars
	if len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' {
		return true
	}
	// Numeric ID
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
