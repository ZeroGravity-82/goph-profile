package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/ZeroGravity-82/goph-profile/internal/observability"
)

// TestWithMetrics_RecordsHandledHTTPRequest проверяет запись метрик после обработки HTTP-запроса.
func TestWithMetrics_RecordsHandledHTTPRequest(t *testing.T) {
	// Arrange
	metrics, reader := newHTTPMetricsForTest(t)
	router := chi.NewRouter()
	router.Use(withMetrics(metrics))
	router.Post("/api/v1/avatars/{avatar_id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("created"))
		require.NoError(t, err)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/avatars/"+testAvatarID.String(), nil)
	response := httptest.NewRecorder()

	// Act
	router.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, "created", response.Body.String())

	resourceMetrics := collectHTTPMetrics(t, reader)
	requests := int64SumDataPoint(t, resourceMetrics, "http_server_requests_total")
	assert.Equal(t, int64(1), requests.Value)
	assert.Equal(t, http.MethodPost, metricAttrString(t, requests.Attributes, "http_request_method"))
	assert.Equal(t, "/api/v1/avatars/{avatar_id}", metricAttrString(t, requests.Attributes, "http_route"))
	assert.Equal(t, int64(http.StatusCreated), metricAttrInt64(t, requests.Attributes, "http_response_status_code"))

	duration := float64HistogramDataPoint(t, resourceMetrics, "http_server_request_duration_seconds")
	assert.Equal(t, uint64(1), duration.Count)

	responseSize := int64HistogramDataPoint(t, resourceMetrics, "http_server_response_size_bytes")
	assert.Equal(t, uint64(1), responseSize.Count)
	assert.Equal(t, int64(len("created")), responseSize.Sum)

	activeRequests := int64SumDataPoint(t, resourceMetrics, "http_server_requests_active")
	assert.Equal(t, int64(0), activeRequests.Value)
	assert.Equal(t, http.MethodPost, metricAttrString(t, activeRequests.Attributes, "http_request_method"))
}

// TestWithMetrics_RecordsDefaultOKStatus проверяет запись 200 OK, если хендлер не записал статус ответа.
func TestWithMetrics_RecordsDefaultOKStatus(t *testing.T) {
	// Arrange
	metrics, reader := newHTTPMetricsForTest(t)
	handler := withMetrics(metrics)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/avatars", nil)
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	requests := int64SumDataPoint(t, collectHTTPMetrics(t, reader), "http_server_requests_total")
	assert.Equal(t, int64(1), requests.Value)
	assert.Equal(t, http.MethodGet, metricAttrString(t, requests.Attributes, "http_request_method"))
	assert.Equal(t, "unknown", metricAttrString(t, requests.Attributes, "http_route"))
	assert.Equal(t, int64(http.StatusOK), metricAttrInt64(t, requests.Attributes, "http_response_status_code"))
}

func newHTTPMetricsForTest(t *testing.T) (*observability.HTTPRequestMetrics, *sdkmetric.ManualReader) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previousMeterProvider := otel.GetMeterProvider()
	otel.SetMeterProvider(meterProvider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previousMeterProvider)
		require.NoError(t, meterProvider.Shutdown(context.Background()))
	})

	metrics, err := observability.NewHTTPRequestMetrics()
	require.NoError(t, err)

	return metrics, reader
}

func collectHTTPMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &resourceMetrics))

	return resourceMetrics
}

func int64SumDataPoint(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	name string,
) metricdata.DataPoint[int64] {
	t.Helper()

	metric := findMetric(t, resourceMetrics, name)
	sum, ok := metric.Data.(metricdata.Sum[int64])
	require.Truef(t, ok, "metric %q has unexpected data type %T", name, metric.Data)
	require.Len(t, sum.DataPoints, 1)

	return sum.DataPoints[0]
}

func float64HistogramDataPoint(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	name string,
) metricdata.HistogramDataPoint[float64] {
	t.Helper()

	metric := findMetric(t, resourceMetrics, name)
	histogram, ok := metric.Data.(metricdata.Histogram[float64])
	require.Truef(t, ok, "metric %q has unexpected data type %T", name, metric.Data)
	require.Len(t, histogram.DataPoints, 1)

	return histogram.DataPoints[0]
}

func int64HistogramDataPoint(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	name string,
) metricdata.HistogramDataPoint[int64] {
	t.Helper()

	metric := findMetric(t, resourceMetrics, name)
	histogram, ok := metric.Data.(metricdata.Histogram[int64])
	require.Truef(t, ok, "metric %q has unexpected data type %T", name, metric.Data)
	require.Len(t, histogram.DataPoints, 1)

	return histogram.DataPoints[0]
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

func metricAttrString(t *testing.T, attrs attribute.Set, key string) string {
	t.Helper()

	value, ok := attrs.Value(attribute.Key(key))
	require.Truef(t, ok, "metric attribute %q not found", key)

	return value.AsString()
}

func metricAttrInt64(t *testing.T, attrs attribute.Set, key string) int64 {
	t.Helper()

	value, ok := attrs.Value(attribute.Key(key))
	require.Truef(t, ok, "metric attribute %q not found", key)

	return value.AsInt64()
}
