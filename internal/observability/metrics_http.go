package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// HTTPRequestMetrics содержит базовые метрики HTTP-запросов.
//
// requests считает завершенные HTTP-запросы с разбивкой по методу, маршруту и коду ответа.
//
// duration хранит распределение длительности HTTP-запросов.
//
// responseSizeBytes хранит распределение размеров HTTP-ответов.
//
// activeRequests хранит текущее количество HTTP-запросов, находящихся в обработке.
type HTTPRequestMetrics struct {
	requests          metric.Int64Counter
	duration          metric.Float64Histogram
	responseSizeBytes metric.Int64Histogram
	activeRequests    metric.Int64UpDownCounter
}

// NewHTTPRequestMetrics создает метрики HTTP-запросов.
func NewHTTPRequestMetrics() (*HTTPRequestMetrics, error) {
	meter := otel.Meter(metricInstrumentationName)

	requests, err := meter.Int64Counter(
		"http_server_requests_total",
		metric.WithDescription("Total number of HTTP requests."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP requests metric: %w", err)
	}
	duration, err := meter.Float64Histogram(
		"http_server_request_duration_seconds",
		metric.WithDescription("HTTP request duration."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request duration metric: %w", err)
	}
	responseSizeBytes, err := meter.Int64Histogram(
		"http_server_response_size_bytes",
		metric.WithDescription("HTTP response size."),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP response size metric: %w", err)
	}
	activeRequests, err := meter.Int64UpDownCounter(
		"http_server_requests_active",
		metric.WithDescription("Number of HTTP requests currently being handled."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active HTTP requests metric: %w", err)
	}

	return &HTTPRequestMetrics{
		requests:          requests,
		duration:          duration,
		responseSizeBytes: responseSizeBytes,
		activeRequests:    activeRequests,
	}, nil
}

// RecordHTTPRequest записывает результат обработки HTTP-запроса.
func (m *HTTPRequestMetrics) RecordHTTPRequest(
	ctx context.Context,
	method string,
	route string,
	status int,
	duration time.Duration,
	responseSizeBytes int,
) {
	attrs := metric.WithAttributes(
		attribute.String("http_request_method", method),
		attribute.String("http_route", route),
		attribute.Int("http_response_status_code", status),
	)

	m.requests.Add(ctx, 1, attrs)
	m.duration.Record(ctx, duration.Seconds(), attrs)
	m.responseSizeBytes.Record(ctx, int64(responseSizeBytes), attrs)
}

// RecordHTTPRequestStarted записывает начало обработки HTTP-запроса.
func (m *HTTPRequestMetrics) RecordHTTPRequestStarted(ctx context.Context, method string) {
	m.activeRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("http_request_method", method)))
}

// RecordHTTPRequestFinished записывает завершение обработки HTTP-запроса.
func (m *HTTPRequestMetrics) RecordHTTPRequestFinished(ctx context.Context, method string) {
	m.activeRequests.Add(ctx, -1, metric.WithAttributes(attribute.String("http_request_method", method)))
}
