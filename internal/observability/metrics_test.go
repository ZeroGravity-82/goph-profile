package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestHTTPRequestMetrics проверяет запись базовых HTTP-метрик.
func TestHTTPRequestMetrics(t *testing.T) {
	// Arrange
	reader := newManualMetricReader(t)
	metrics, err := NewHTTPRequestMetrics()
	require.NoError(t, err)

	// Act
	metrics.RecordHTTPRequestStarted(context.Background(), "GET")
	metrics.RecordHTTPRequest(context.Background(), "GET", "/api/v1/avatar", 200, 150*time.Millisecond, 42)
	metrics.RecordHTTPRequestFinished(context.Background(), "GET")

	// Assert
	resourceMetrics := collectMetrics(t, reader)

	requests := int64SumDataPoint(t, resourceMetrics, "http_server_requests_total", map[string]string{
		"http_request_method":       "GET",
		"http_route":                "/api/v1/avatar",
		"http_response_status_code": "200",
	})
	assert.Equal(t, int64(1), requests.Value)

	duration := float64HistogramDataPoint(t, resourceMetrics, "http_server_request_duration_seconds", map[string]string{
		"http_request_method":       "GET",
		"http_route":                "/api/v1/avatar",
		"http_response_status_code": "200",
	})
	assert.Equal(t, uint64(1), duration.Count)

	responseSize := int64HistogramDataPoint(t, resourceMetrics, "http_server_response_size_bytes", map[string]string{
		"http_request_method":       "GET",
		"http_route":                "/api/v1/avatar",
		"http_response_status_code": "200",
	})
	assert.Equal(t, uint64(1), responseSize.Count)
	assert.Equal(t, int64(42), responseSize.Sum)

	activeRequests := int64SumDataPoint(t, resourceMetrics, "http_server_requests_active", map[string]string{
		"http_request_method": "GET",
	})
	assert.Equal(t, int64(0), activeRequests.Value)
}

// TestAvatarUseCaseMetrics проверяет запись метрик пользовательских сценариев аватарок.
func TestAvatarUseCaseMetrics(t *testing.T) {
	// Arrange
	reader := newManualMetricReader(t)
	metrics, err := NewAvatarUseCaseMetrics()
	require.NoError(t, err)

	// Act
	metrics.RecordAvatarUpload(context.Background(), "success", "created", "image/png", 123)
	metrics.RecordAvatarUseCaseAction(context.Background(), "delete_by_id", "error", "avatar_not_found")

	// Assert
	resourceMetrics := collectMetrics(t, reader)

	uploads := int64SumDataPoint(t, resourceMetrics, "avatars_uploads_total", map[string]string{
		"status":    "success",
		"reason":    "created",
		"mime_type": "image/png",
	})
	assert.Equal(t, int64(1), uploads.Value)

	uploadSize := int64HistogramDataPoint(t, resourceMetrics, "avatars_upload_size_bytes", map[string]string{
		"status":    "success",
		"reason":    "created",
		"mime_type": "image/png",
	})
	assert.Equal(t, uint64(1), uploadSize.Count)
	assert.Equal(t, int64(123), uploadSize.Sum)

	actions := int64SumDataPoint(t, resourceMetrics, "avatar_api_actions_total", map[string]string{
		"action": "delete_by_id",
		"status": "error",
		"reason": "avatar_not_found",
	})
	assert.Equal(t, int64(1), actions.Value)

	failures := int64SumDataPoint(t, resourceMetrics, "avatar_api_failures_total", map[string]string{
		"action": "delete_by_id",
		"status": "error",
		"reason": "avatar_not_found",
	})
	assert.Equal(t, int64(1), failures.Value)
}

// TestAvatarAsyncMetrics проверяет запись метрик асинхронной обработки и удаления аватарок.
func TestAvatarAsyncMetrics(t *testing.T) {
	// Arrange
	reader := newManualMetricReader(t)
	metrics, err := NewAvatarAsyncMetrics()
	require.NoError(t, err)

	// Act
	metrics.RecordAvatarProcessing(context.Background(), "success", "ready", 250*time.Millisecond, 1024, 100, 300)
	metrics.RecordAvatarDeletion(context.Background(), "success", "deleted", 100*time.Millisecond, 3)

	// Assert
	resourceMetrics := collectMetrics(t, reader)

	processingJob := int64SumDataPoint(t, resourceMetrics, "avatar_worker_jobs_total", map[string]string{
		"operation": "process",
		"status":    "success",
		"reason":    "ready",
	})
	assert.Equal(t, int64(1), processingJob.Value)

	deletionJob := int64SumDataPoint(t, resourceMetrics, "avatar_worker_jobs_total", map[string]string{
		"operation": "delete",
		"status":    "success",
		"reason":    "deleted",
	})
	assert.Equal(t, int64(1), deletionJob.Value)

	originalSize := int64HistogramDataPoint(t, resourceMetrics, "avatar_worker_original_size_bytes", map[string]string{
		"operation": "process",
		"status":    "success",
		"reason":    "ready",
	})
	assert.Equal(t, uint64(1), originalSize.Count)
	assert.Equal(t, int64(1024), originalSize.Sum)

	thumb100Size := int64HistogramDataPoint(t, resourceMetrics, "avatar_worker_thumbnail_size_bytes", map[string]string{
		"size":   "100x100",
		"status": "success",
		"reason": "ready",
	})
	assert.Equal(t, uint64(1), thumb100Size.Count)
	assert.Equal(t, int64(100), thumb100Size.Sum)

	thumb300Size := int64HistogramDataPoint(t, resourceMetrics, "avatar_worker_thumbnail_size_bytes", map[string]string{
		"size":   "300x300",
		"status": "success",
		"reason": "ready",
	})
	assert.Equal(t, uint64(1), thumb300Size.Count)
	assert.Equal(t, int64(300), thumb300Size.Sum)

	deletedObjects := int64SumDataPoint(t, resourceMetrics, "avatar_worker_deleted_objects_total", map[string]string{
		"operation": "delete",
		"status":    "success",
		"reason":    "deleted",
	})
	assert.Equal(t, int64(3), deletedObjects.Value)
}

func newManualMetricReader(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previousMeterProvider := otel.GetMeterProvider()
	otel.SetMeterProvider(meterProvider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previousMeterProvider)
		require.NoError(t, meterProvider.Shutdown(context.Background()))
	})

	return reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &resourceMetrics))

	return resourceMetrics
}

func int64SumDataPoint(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	name string,
	attrs map[string]string,
) metricdata.DataPoint[int64] {
	t.Helper()

	metric := findMetric(t, resourceMetrics, name)
	sum, ok := metric.Data.(metricdata.Sum[int64])
	require.Truef(t, ok, "metric %q has unexpected data type %T", name, metric.Data)

	for _, point := range sum.DataPoints {
		if metricAttrsContain(point.Attributes, attrs) {
			return point
		}
	}
	require.FailNowf(t, "metric data point not found", "metric %q with attrs %v not found", name, attrs)

	return metricdata.DataPoint[int64]{}
}

func float64HistogramDataPoint(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	name string,
	attrs map[string]string,
) metricdata.HistogramDataPoint[float64] {
	t.Helper()

	metric := findMetric(t, resourceMetrics, name)
	histogram, ok := metric.Data.(metricdata.Histogram[float64])
	require.Truef(t, ok, "metric %q has unexpected data type %T", name, metric.Data)

	for _, point := range histogram.DataPoints {
		if metricAttrsContain(point.Attributes, attrs) {
			return point
		}
	}
	require.FailNowf(t, "metric data point not found", "metric %q with attrs %v not found", name, attrs)

	return metricdata.HistogramDataPoint[float64]{}
}

func int64HistogramDataPoint(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	name string,
	attrs map[string]string,
) metricdata.HistogramDataPoint[int64] {
	t.Helper()

	metric := findMetric(t, resourceMetrics, name)
	histogram, ok := metric.Data.(metricdata.Histogram[int64])
	require.Truef(t, ok, "metric %q has unexpected data type %T", name, metric.Data)

	for _, point := range histogram.DataPoints {
		if metricAttrsContain(point.Attributes, attrs) {
			return point
		}
	}
	require.FailNowf(t, "metric data point not found", "metric %q with attrs %v not found", name, attrs)

	return metricdata.HistogramDataPoint[int64]{}
}

func findMetric(t *testing.T, resourceMetrics metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()

	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, item := range scopeMetrics.Metrics {
			if item.Name == name {
				return item
			}
		}
	}
	require.FailNowf(t, "metric not found", "metric %q not found", name)

	return metricdata.Metrics{}
}

func metricAttrsContain(attrs attribute.Set, want map[string]string) bool {
	for key, wantValue := range want {
		value, ok := attrs.Value(attribute.Key(key))
		if !ok || value.String() != wantValue {
			return false
		}
	}

	return true
}
