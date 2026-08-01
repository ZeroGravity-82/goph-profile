package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// TestStartSpan проверяет создание span трассировки для операции PostgreSQL.
func TestStartSpan(t *testing.T) {
	// Arrange
	exporter := newTraceExporter(t)

	// Act
	_, span := startSpan(context.Background(), "postgres.avatar.get_by_id", "SELECT", "avatar")
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "postgres.avatar.get_by_id", spans[0].Name)
	assert.Equal(t, trace.SpanKindClient, spans[0].SpanKind)
	assert.Equal(t, traceInstrumentationName, spans[0].InstrumentationScope.Name)
	assert.Contains(t, spans[0].Attributes, attribute.String("db.system.name", "postgresql"))
	assert.Contains(t, spans[0].Attributes, attribute.String("db.operation.name", "SELECT"))
	assert.Contains(t, spans[0].Attributes, attribute.String("db.collection.name", "avatar"))
}

// TestStartSpan_WithoutCollection проверяет span PostgreSQL без атрибута таблицы.
func TestStartSpan_WithoutCollection(t *testing.T) {
	// Arrange
	exporter := newTraceExporter(t)

	// Act
	_, span := startSpan(context.Background(), "postgres.transaction", "TRANSACTION", "")
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "postgres.transaction", spans[0].Name)
	assert.NotContains(t, spans[0].Attributes, attribute.String("db.collection.name", ""))
}

// TestRecordSpanError проверяет запись ошибки в span PostgreSQL.
func TestRecordSpanError(t *testing.T) {
	// Arrange
	exporter := newTraceExporter(t)
	_, span := startSpan(context.Background(), "postgres.user.update", "UPDATE", "app_user")

	// Act
	recordSpanError(span, errors.New("update failed"))
	span.End()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
	assert.Equal(t, "update failed", spans[0].Status.Description)
	require.Len(t, spans[0].Events, 1)
	assert.Equal(t, "exception", spans[0].Events[0].Name)
}

func newTraceExporter(t *testing.T) *tracetest.InMemoryExporter {
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
