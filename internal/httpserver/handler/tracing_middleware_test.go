package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TestWithHTTPRouteTracing_RecordsRoutePattern проверяет запись шаблона chi-маршрута в HTTP span.
func TestWithHTTPRouteTracing_RecordsRoutePattern(t *testing.T) {
	// Arrange
	exporter := newHTTPTraceExporter(t)
	router := chi.NewRouter()
	router.Use(withHTTPRouteTracing)
	router.Get("/api/v1/avatars/{avatar_id}/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := otelhttp.NewHandler(router, "goph-profile.http")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+testAvatarID.String()+"/metadata", nil)
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, http.StatusOK, response.Code)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "GET /api/v1/avatars/{avatar_id}/metadata", spans[0].Name)
	assert.Equal(t, oteltrace.SpanKindServer, spans[0].SpanKind)
	assert.Contains(t, spans[0].Attributes, attribute.String("http.route", "/api/v1/avatars/{avatar_id}/metadata"))
}

// TestWithHTTPRouteTracing_SkipsUnknownRoute проверяет, что неизвестный маршрут не записывается в HTTP span.
func TestWithHTTPRouteTracing_SkipsUnknownRoute(t *testing.T) {
	// Arrange
	exporter := newHTTPTraceExporter(t)
	router := chi.NewRouter()
	router.Use(withHTTPRouteTracing)
	router.Get("/api/v1/avatars/{avatar_id}/metadata", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := otelhttp.NewHandler(router, "goph-profile.http")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	response := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, http.MethodGet, spans[0].Name)
	assert.NotContains(t, spans[0].Attributes, attribute.String("http.route", unknownHTTPRoute))
}

func newHTTPTraceExporter(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previousTracerProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(tracerProvider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		require.NoError(t, tracerProvider.Shutdown(context.Background()))
	})

	return exporter
}
