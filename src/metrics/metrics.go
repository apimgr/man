// Package metrics implements Prometheus-compatible metrics for casman.
// See AI.md PART 21 for details.
package metrics

import (
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "casman"

// Config holds metrics configuration.
type Config struct {
	Enabled        bool
	Endpoint       string
	Token          string
	IncludeSystem  bool
	IncludeRuntime bool
}

// DefaultConfig returns the default metrics configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		Endpoint:       "/metrics",
		IncludeSystem:  true,
		IncludeRuntime: true,
	}
}

// Metrics holds all Prometheus metrics.
type Metrics struct {
	config    Config
	registry  *prometheus.Registry
	startTime time.Time
	mu        sync.RWMutex

	// Version info
	version   string
	commit    string
	buildDate string
	goVersion string

	// Application metrics
	appInfo            *prometheus.GaugeVec
	appUptime          prometheus.Gauge
	appStartTimestamp  prometheus.Gauge

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
	httpActiveRequests  prometheus.Gauge

	// Database metrics
	dbQueriesTotal     *prometheus.CounterVec
	dbQueryDuration    *prometheus.HistogramVec
	dbConnectionsOpen  prometheus.Gauge
	dbConnectionsInUse prometheus.Gauge
	dbErrorsTotal      *prometheus.CounterVec

	// Cache metrics
	cacheHitsTotal      *prometheus.CounterVec
	cacheMissesTotal    *prometheus.CounterVec
	cacheEvictionsTotal *prometheus.CounterVec
	cacheSize           *prometheus.GaugeVec
	cacheBytes          *prometheus.GaugeVec

	// Auth metrics
	authAttemptsTotal  *prometheus.CounterVec
	authSessionsActive prometheus.Gauge

	// System metrics
	systemCPUUsage     prometheus.Gauge
	systemMemoryUsage  prometheus.Gauge
	systemMemoryUsed   prometheus.Gauge
	systemMemoryTotal  prometheus.Gauge
	systemDiskUsage    *prometheus.GaugeVec
	systemDiskUsed     *prometheus.GaugeVec
	systemDiskTotal    *prometheus.GaugeVec
}

// New creates a new Metrics instance.
func New(cfg Config, version, commit, buildDate string) *Metrics {
	m := &Metrics{
		config:    cfg,
		registry:  prometheus.NewRegistry(),
		startTime: time.Now(),
		version:   version,
		commit:    commit,
		buildDate: buildDate,
		goVersion: runtime.Version(),
	}

	m.initMetrics()

	// Register Go runtime metrics if enabled
	if cfg.IncludeRuntime {
		m.registry.MustRegister(collectors.NewGoCollector())
	}

	// Set app info
	m.appInfo.WithLabelValues(m.version, m.commit, m.buildDate, m.goVersion).Set(1)
	m.appStartTimestamp.Set(float64(m.startTime.Unix()))

	return m
}

// initMetrics initializes all metric collectors.
func (m *Metrics) initMetrics() {
	// HTTP duration buckets
	durationBuckets := []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	// Size buckets
	sizeBuckets := []float64{100, 1000, 10000, 100000, 1000000, 10000000}
	// DB query buckets
	dbBuckets := []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1}

	// Application metrics
	m.appInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "app_info",
			Help:      "Application build information",
		},
		[]string{"version", "commit", "build_date", "go_version"},
	)

	m.appUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "app_uptime_seconds",
		Help:      "Seconds since application start",
	})

	m.appStartTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "app_start_timestamp",
		Help:      "Unix timestamp when application started",
	})

	// HTTP metrics
	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests processed",
		},
		[]string{"method", "path", "status"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Request latency distribution",
			Buckets:   durationBuckets,
		},
		[]string{"method", "path"},
	)

	m.httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_size_bytes",
			Help:      "Request body size distribution",
			Buckets:   sizeBuckets,
		},
		[]string{"method", "path"},
	)

	m.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_response_size_bytes",
			Help:      "Response body size distribution",
			Buckets:   sizeBuckets,
		},
		[]string{"method", "path"},
	)

	m.httpActiveRequests = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "http_active_requests",
		Help:      "Number of requests currently being processed",
	})

	// Database metrics
	m.dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "db_queries_total",
			Help:      "Total database queries",
		},
		[]string{"operation", "table"},
	)

	m.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "db_query_duration_seconds",
			Help:      "Query latency distribution",
			Buckets:   dbBuckets,
		},
		[]string{"operation", "table"},
	)

	m.dbConnectionsOpen = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "db_connections_open",
		Help:      "Number of open connections in pool",
	})

	m.dbConnectionsInUse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "db_connections_in_use",
		Help:      "Number of connections actively in use",
	})

	m.dbErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "db_errors_total",
			Help:      "Total database errors",
		},
		[]string{"operation", "error_type"},
	)

	// Cache metrics
	m.cacheHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_hits_total",
			Help:      "Total cache hits",
		},
		[]string{"cache"},
	)

	m.cacheMissesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_misses_total",
			Help:      "Total cache misses",
		},
		[]string{"cache"},
	)

	m.cacheEvictionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_evictions_total",
			Help:      "Total cache evictions",
		},
		[]string{"cache"},
	)

	m.cacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cache_size",
			Help:      "Current number of items in cache",
		},
		[]string{"cache"},
	)

	m.cacheBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cache_bytes",
			Help:      "Current cache size in bytes",
		},
		[]string{"cache"},
	)

	// Auth metrics
	m.authAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "auth_attempts_total",
			Help:      "Total authentication attempts",
		},
		[]string{"method", "status"},
	)

	m.authSessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "auth_sessions_active",
		Help:      "Number of active sessions",
	})

	// System metrics
	m.systemCPUUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "system_cpu_usage_percent",
		Help:      "Current CPU usage percentage (0-100)",
	})

	m.systemMemoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "system_memory_usage_percent",
		Help:      "Current memory usage percentage (0-100)",
	})

	m.systemMemoryUsed = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "system_memory_used_bytes",
		Help:      "Memory currently in use",
	})

	m.systemMemoryTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "system_memory_total_bytes",
		Help:      "Total system memory",
	})

	m.systemDiskUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "system_disk_usage_percent",
			Help:      "Disk usage percentage for data directory",
		},
		[]string{"path"},
	)

	m.systemDiskUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "system_disk_used_bytes",
			Help:      "Disk space used",
		},
		[]string{"path"},
	)

	m.systemDiskTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "system_disk_total_bytes",
			Help:      "Total disk space",
		},
		[]string{"path"},
	)

	// Register all metrics
	m.registry.MustRegister(
		m.appInfo,
		m.appUptime,
		m.appStartTimestamp,
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestSize,
		m.httpResponseSize,
		m.httpActiveRequests,
		m.dbQueriesTotal,
		m.dbQueryDuration,
		m.dbConnectionsOpen,
		m.dbConnectionsInUse,
		m.dbErrorsTotal,
		m.cacheHitsTotal,
		m.cacheMissesTotal,
		m.cacheEvictionsTotal,
		m.cacheSize,
		m.cacheBytes,
		m.authAttemptsTotal,
		m.authSessionsActive,
	)

	// Register system metrics if enabled
	if m.config.IncludeSystem {
		m.registry.MustRegister(
			m.systemCPUUsage,
			m.systemMemoryUsage,
			m.systemMemoryUsed,
			m.systemMemoryTotal,
			m.systemDiskUsage,
			m.systemDiskUsed,
			m.systemDiskTotal,
		)
	}
}

// Handler returns the HTTP handler for the /metrics endpoint.
func (m *Metrics) Handler() http.Handler {
	// Create handler that updates uptime before serving
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check token authentication if configured
		if m.config.Token != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + m.config.Token
			if auth != expected {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Update uptime
		m.appUptime.Set(time.Since(m.startTime).Seconds())

		// Serve metrics
		promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})
}

// RecordHTTPRequest records metrics for an HTTP request.
func (m *Metrics) RecordHTTPRequest(method, path string, status int, duration time.Duration, reqSize, respSize int64) {
	// Normalize path to reduce cardinality
	normalizedPath := normalizePath(path)

	m.httpRequestsTotal.WithLabelValues(method, normalizedPath, strconv.Itoa(status)).Inc()
	m.httpRequestDuration.WithLabelValues(method, normalizedPath).Observe(duration.Seconds())
	m.httpRequestSize.WithLabelValues(method, normalizedPath).Observe(float64(reqSize))
	m.httpResponseSize.WithLabelValues(method, normalizedPath).Observe(float64(respSize))
}

// IncActiveRequests increments the active requests gauge.
func (m *Metrics) IncActiveRequests() {
	m.httpActiveRequests.Inc()
}

// DecActiveRequests decrements the active requests gauge.
func (m *Metrics) DecActiveRequests() {
	m.httpActiveRequests.Dec()
}

// RecordDBQuery records metrics for a database query.
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration) {
	m.dbQueriesTotal.WithLabelValues(operation, table).Inc()
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordDBError records a database error.
func (m *Metrics) RecordDBError(operation, errorType string) {
	m.dbErrorsTotal.WithLabelValues(operation, errorType).Inc()
}

// SetDBConnections sets the database connection metrics.
func (m *Metrics) SetDBConnections(open, inUse int) {
	m.dbConnectionsOpen.Set(float64(open))
	m.dbConnectionsInUse.Set(float64(inUse))
}

// RecordCacheHit records a cache hit.
func (m *Metrics) RecordCacheHit(cache string) {
	m.cacheHitsTotal.WithLabelValues(cache).Inc()
}

// RecordCacheMiss records a cache miss.
func (m *Metrics) RecordCacheMiss(cache string) {
	m.cacheMissesTotal.WithLabelValues(cache).Inc()
}

// RecordCacheEviction records a cache eviction.
func (m *Metrics) RecordCacheEviction(cache string) {
	m.cacheEvictionsTotal.WithLabelValues(cache).Inc()
}

// SetCacheSize sets the cache size metrics.
func (m *Metrics) SetCacheSize(cache string, items int, bytes int64) {
	m.cacheSize.WithLabelValues(cache).Set(float64(items))
	m.cacheBytes.WithLabelValues(cache).Set(float64(bytes))
}

// RecordAuthAttempt records an authentication attempt.
func (m *Metrics) RecordAuthAttempt(method, status string) {
	m.authAttemptsTotal.WithLabelValues(method, status).Inc()
}

// SetActiveSessions sets the active sessions gauge.
func (m *Metrics) SetActiveSessions(count int) {
	m.authSessionsActive.Set(float64(count))
}

// SetSystemMetrics sets system resource metrics.
func (m *Metrics) SetSystemMetrics(cpuPercent, memPercent float64, memUsed, memTotal uint64) {
	if !m.config.IncludeSystem {
		return
	}
	m.systemCPUUsage.Set(cpuPercent)
	m.systemMemoryUsage.Set(memPercent)
	m.systemMemoryUsed.Set(float64(memUsed))
	m.systemMemoryTotal.Set(float64(memTotal))
}

// SetDiskMetrics sets disk usage metrics.
func (m *Metrics) SetDiskMetrics(path string, usagePercent float64, used, total uint64) {
	if !m.config.IncludeSystem {
		return
	}
	m.systemDiskUsage.WithLabelValues(path).Set(usagePercent)
	m.systemDiskUsed.WithLabelValues(path).Set(float64(used))
	m.systemDiskTotal.WithLabelValues(path).Set(float64(total))
}

// normalizePath reduces path cardinality by replacing IDs with :id.
func normalizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Replace UUIDs
		if isUUID(part) {
			parts[i] = ":id"
			continue
		}
		// Replace numeric IDs
		if isNumericID(part) {
			parts[i] = ":id"
			continue
		}
	}
	return strings.Join(parts, "/")
}

// isUUID checks if a string looks like a UUID.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Check format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !isHexDigit(byte(c)) {
				return false
			}
		}
	}
	return true
}

// isHexDigit checks if a byte is a hex digit.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// isNumericID checks if a string is a numeric ID (all digits, 1-20 chars).
func isNumericID(s string) bool {
	if len(s) == 0 || len(s) > 20 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
