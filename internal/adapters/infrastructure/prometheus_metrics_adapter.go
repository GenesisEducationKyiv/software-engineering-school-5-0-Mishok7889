package infrastructure

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"weatherapi.app/pkg/logger"
)

// PrometheusMetricsAdapter wraps existing metrics system with Prometheus export
type PrometheusMetricsAdapter struct {
	metricsCollector *MetricsCollectorAdapter
	registry         *prometheus.Registry
	logger           *logger.Logger

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Weather metrics
	weatherRequestsTotal    *prometheus.CounterVec
	weatherRequestDuration  *prometheus.HistogramVec
	weatherProviderFailures *prometheus.CounterVec

	// Cache metrics
	cacheOperationsTotal *prometheus.CounterVec
	cacheHitRatio        prometheus.Gauge
	cacheSizeBytes       prometheus.Gauge

	// Database metrics
	dbConnectionsActive prometheus.Gauge
	dbQueryDuration     *prometheus.HistogramVec
	dbQueryTotal        *prometheus.CounterVec

	// Message broker metrics
	messageBrokerPublished *prometheus.CounterVec
	messageBrokerConsumed  *prometheus.CounterVec
	messageBrokerErrors    *prometheus.CounterVec

	// Business metrics
	subscriptionsActive prometheus.Gauge
	subscriptionsTotal  *prometheus.CounterVec
	emailsSent          *prometheus.CounterVec
}

// PrometheusConfig holds configuration for Prometheus metrics
type PrometheusConfig struct {
	ServiceName      string
	ServiceVersion   string
	MetricsCollector *MetricsCollectorAdapter
	Logger           *logger.Logger
}

// NewPrometheusMetricsAdapter creates a new Prometheus metrics adapter
func NewPrometheusMetricsAdapter(config PrometheusConfig) *PrometheusMetricsAdapter {
	registry := prometheus.NewRegistry()

	adapter := &PrometheusMetricsAdapter{
		metricsCollector: config.MetricsCollector,
		registry:         registry,
		logger:           config.Logger,
	}

	adapter.initializeMetrics(config.ServiceName, config.ServiceVersion)
	adapter.registerMetrics()

	return adapter
}

func (p *PrometheusMetricsAdapter) createCounterVec(name, help string, commonLabels prometheus.Labels, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        name,
			Help:        help,
			ConstLabels: commonLabels,
		},
		labels,
	)
}

func (p *PrometheusMetricsAdapter) createHistogramVec(name, help string, commonLabels prometheus.Labels, labels []string, buckets []float64) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        name,
			Help:        help,
			ConstLabels: commonLabels,
			Buckets:     buckets,
		},
		labels,
	)
}

func (p *PrometheusMetricsAdapter) createGauge(name, help string, commonLabels prometheus.Labels) prometheus.Gauge {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        name,
			Help:        help,
			ConstLabels: commonLabels,
		},
	)
}

// initializeMetrics creates all Prometheus metrics
func (p *PrometheusMetricsAdapter) initializeMetrics(serviceName, serviceVersion string) {
	commonLabels := prometheus.Labels{
		"service": serviceName,
		"version": serviceVersion,
	}

	// HTTP metrics
	p.httpRequestsTotal = p.createCounterVec(
		"http_requests_total",
		"Total number of HTTP requests",
		commonLabels,
		[]string{"method", "endpoint", "status_code"},
	)

	p.httpRequestDuration = p.createHistogramVec(
		"http_request_duration_seconds",
		"HTTP request duration in seconds",
		commonLabels,
		[]string{"method", "endpoint"},
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	)

	p.httpResponseSize = p.createHistogramVec(
		"http_response_size_bytes",
		"HTTP response size in bytes",
		commonLabels,
		[]string{"method", "endpoint"},
		prometheus.ExponentialBuckets(100, 10, 8),
	)

	// Weather metrics
	p.weatherRequestsTotal = p.createCounterVec(
		"weather_requests_total",
		"Total number of weather API requests",
		commonLabels,
		[]string{"provider", "city", "status"},
	)

	p.weatherRequestDuration = p.createHistogramVec(
		"weather_request_duration_seconds",
		"Weather API request duration in seconds",
		commonLabels,
		[]string{"provider"},
		[]float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	)

	p.weatherProviderFailures = p.createCounterVec(
		"weather_provider_failures_total",
		"Total number of weather provider failures",
		commonLabels,
		[]string{"provider", "error_type"},
	)

	// Cache metrics
	p.cacheOperationsTotal = p.createCounterVec(
		"cache_operations_total",
		"Total number of cache operations",
		commonLabels,
		[]string{"operation", "result"},
	)

	p.cacheHitRatio = p.createGauge(
		"cache_hit_ratio",
		"Cache hit ratio (0-1)",
		commonLabels,
	)

	p.cacheSizeBytes = p.createGauge(
		"cache_size_bytes",
		"Current cache size in bytes",
		commonLabels,
	)

	// Database metrics
	p.dbConnectionsActive = p.createGauge(
		"database_connections_active",
		"Number of active database connections",
		commonLabels,
	)

	p.dbQueryDuration = p.createHistogramVec(
		"database_query_duration_seconds",
		"Database query duration in seconds",
		commonLabels,
		[]string{"operation", "table"},
		[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1},
	)

	p.dbQueryTotal = p.createCounterVec(
		"database_queries_total",
		"Total number of database queries",
		commonLabels,
		[]string{"operation", "table", "status"},
	)

	// Message broker metrics
	p.messageBrokerPublished = p.createCounterVec(
		"message_broker_published_total",
		"Total number of published messages",
		commonLabels,
		[]string{"topic", "status"},
	)

	p.messageBrokerConsumed = p.createCounterVec(
		"message_broker_consumed_total",
		"Total number of consumed messages",
		commonLabels,
		[]string{"topic", "consumer_group", "status"},
	)

	p.messageBrokerErrors = p.createCounterVec(
		"message_broker_errors_total",
		"Total number of message broker errors",
		commonLabels,
		[]string{"operation", "error_type"},
	)

	// Business metrics
	p.subscriptionsActive = p.createGauge(
		"subscriptions_active",
		"Number of active subscriptions",
		commonLabels,
	)

	p.subscriptionsTotal = p.createCounterVec(
		"subscriptions_total",
		"Total number of subscription operations",
		commonLabels,
		[]string{"operation", "frequency"},
	)

	p.emailsSent = p.createCounterVec(
		"emails_sent_total",
		"Total number of emails sent",
		commonLabels,
		[]string{"type", "status"},
	)
}

// registerMetrics registers all metrics with the Prometheus registry
func (p *PrometheusMetricsAdapter) registerMetrics() {
	metrics := []prometheus.Collector{
		p.httpRequestsTotal,
		p.httpRequestDuration,
		p.httpResponseSize,
		p.weatherRequestsTotal,
		p.weatherRequestDuration,
		p.weatherProviderFailures,
		p.cacheOperationsTotal,
		p.cacheHitRatio,
		p.cacheSizeBytes,
		p.dbConnectionsActive,
		p.dbQueryDuration,
		p.dbQueryTotal,
		p.messageBrokerPublished,
		p.messageBrokerConsumed,
		p.messageBrokerErrors,
		p.subscriptionsActive,
		p.subscriptionsTotal,
		p.emailsSent,
	}

	for _, metric := range metrics {
		p.registry.MustRegister(metric)
	}

	p.registry.MustRegister(collectors.NewGoCollector())
	p.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

// HTTP Metrics Methods
func (p *PrometheusMetricsAdapter) RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration, responseSize int64) {
	p.httpRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	p.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	p.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// Weather Metrics Methods
func (p *PrometheusMetricsAdapter) RecordWeatherRequest(provider, city, status string, duration time.Duration) {
	p.weatherRequestsTotal.WithLabelValues(provider, city, status).Inc()
	p.weatherRequestDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

func (p *PrometheusMetricsAdapter) RecordWeatherProviderFailure(provider, errorType string) {
	p.weatherProviderFailures.WithLabelValues(provider, errorType).Inc()
}

// Cache Metrics Methods
func (p *PrometheusMetricsAdapter) RecordCacheOperation(operation, result string) {
	p.cacheOperationsTotal.WithLabelValues(operation, result).Inc()
}

func (p *PrometheusMetricsAdapter) UpdateCacheHitRatio(ratio float64) {
	p.cacheHitRatio.Set(ratio)
}

func (p *PrometheusMetricsAdapter) UpdateCacheSize(sizeBytes float64) {
	p.cacheSizeBytes.Set(sizeBytes)
}

// Database Metrics Methods
func (p *PrometheusMetricsAdapter) RecordDatabaseQuery(operation, table, status string, duration time.Duration) {
	p.dbQueryTotal.WithLabelValues(operation, table, status).Inc()
	p.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

func (p *PrometheusMetricsAdapter) UpdateDatabaseConnections(active float64) {
	p.dbConnectionsActive.Set(active)
}

// Message Broker Metrics Methods
func (p *PrometheusMetricsAdapter) RecordMessagePublished(topic, status string) {
	p.messageBrokerPublished.WithLabelValues(topic, status).Inc()
}

func (p *PrometheusMetricsAdapter) RecordMessageConsumed(topic, consumerGroup, status string) {
	p.messageBrokerConsumed.WithLabelValues(topic, consumerGroup, status).Inc()
}

func (p *PrometheusMetricsAdapter) RecordMessageBrokerError(operation, errorType string) {
	p.messageBrokerErrors.WithLabelValues(operation, errorType).Inc()
}

// Business Metrics Methods
func (p *PrometheusMetricsAdapter) UpdateActiveSubscriptions(count float64) {
	p.subscriptionsActive.Set(count)
}

func (p *PrometheusMetricsAdapter) RecordSubscriptionOperation(operation, frequency string) {
	p.subscriptionsTotal.WithLabelValues(operation, frequency).Inc()
}

func (p *PrometheusMetricsAdapter) RecordEmailSent(emailType, status string) {
	p.emailsSent.WithLabelValues(emailType, status).Inc()
}

// processCacheMetrics processes cache metrics with early returns
func (p *PrometheusMetricsAdapter) processCacheMetrics(legacyMetrics map[string]interface{}) {
	cacheData, ok := legacyMetrics["cache"].(CacheMetrics)
	if !ok {
		return
	}

	p.cacheHitRatio.Set(cacheData.HitRatio)
	p.cacheOperationsTotal.WithLabelValues("get", "hit").Add(float64(cacheData.Hits))
	p.cacheOperationsTotal.WithLabelValues("get", "miss").Add(float64(cacheData.Misses))
}

// processAPIMetrics processes API metrics with early returns
func (p *PrometheusMetricsAdapter) processAPIMetrics(legacyMetrics map[string]interface{}) {
	apiData, ok := legacyMetrics["api"].(map[string]interface{})
	if !ok {
		return
	}

	counters, ok := apiData["counters"].([]interface{})
	if !ok {
		return
	}

	for _, counterInterface := range counters {
		counter, ok := counterInterface.(*CounterMetric)
		if !ok {
			continue
		}

		if counter.Name != "api_requests_total" {
			continue
		}

		method := counter.Labels["method"]
		endpoint := counter.Labels["endpoint"]
		p.httpRequestsTotal.WithLabelValues(method, endpoint, "200").Add(float64(counter.Value))
	}
}

// SyncFromLegacyMetrics updates Prometheus metrics from existing metrics collector
func (p *PrometheusMetricsAdapter) SyncFromLegacyMetrics(ctx context.Context) error {
	if p.metricsCollector == nil {
		return nil
	}

	legacyMetrics, err := p.metricsCollector.GetMetrics(ctx)
	if err != nil {
		p.logger.Error("failed to get legacy metrics", "error", err)
		return err
	}

	p.processCacheMetrics(legacyMetrics)
	p.processAPIMetrics(legacyMetrics)

	return nil
}

// Handler returns the Prometheus HTTP handler
func (p *PrometheusMetricsAdapter) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{
		Registry: p.registry,
	})
}

// GetMetrics returns current metrics in legacy format for compatibility
func (p *PrometheusMetricsAdapter) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	if p.metricsCollector != nil {
		return p.metricsCollector.GetMetrics(ctx)
	}
	return map[string]interface{}{
		"prometheus": "metrics available at /metrics endpoint",
	}, nil
}

// IncrementCounter implements MetricsCollector interface for backward compatibility
func (p *PrometheusMetricsAdapter) IncrementCounter(name string, labels map[string]string) {
	if p.metricsCollector != nil {
		p.metricsCollector.IncrementCounter(name, labels)
	}

	switch name {
	case "api_requests_total":
		method := labels["method"]
		endpoint := labels["endpoint"]
		p.httpRequestsTotal.WithLabelValues(method, endpoint, "unknown").Inc()
	case "api_responses_total":
		endpoint := labels["endpoint"]
		status := labels["status"]
		statusCode := "200"
		if status != "success" {
			statusCode = "500"
		}
		p.httpRequestsTotal.WithLabelValues("unknown", endpoint, statusCode).Inc()
	}
}

// Shutdown gracefully shuts down the metrics adapter
func (p *PrometheusMetricsAdapter) Shutdown(ctx context.Context) error {
	p.logger.Info("shutting down prometheus metrics adapter")
	return nil
}
