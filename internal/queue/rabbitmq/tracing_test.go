package rabbitmq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TestStartConsumeSpanWithExtractedTraceContext проверяет создание span обработки сообщения RabbitMQ после извлечения
// родительского контекста трассировки из заголовков AMQP.
func TestStartConsumeSpanWithExtractedTraceContext(t *testing.T) {
	// Arrange
	exporter := newTraceExporter(t)
	delivery := amqp.Delivery{
		Exchange:   "goph-profile.avatar",
		RoutingKey: "avatar.process",
		MessageId:  "avatar-processing:018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
	}

	// Act
	_, span := startConsumeSpanWithExtractedTraceContext(
		context.Background(),
		delivery,
		"rabbitmq.consume.avatar_processing",
	)
	span.setAvatarID("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")
	span.end()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "rabbitmq.consume.avatar_processing", spans[0].Name)
	assert.Equal(t, oteltrace.SpanKindConsumer, spans[0].SpanKind)
	assert.Equal(t, traceInstrumentationName, spans[0].InstrumentationScope.Name)
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.system", "rabbitmq"))
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.operation.name", "consume"))
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.destination.name", "goph-profile.avatar"))
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.rabbitmq.routing_key", "avatar.process"))
	assert.Contains(t, spans[0].Attributes, attribute.String("avatar.id", "018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"))
}

// TestStartSpan проверяет создание span RabbitMQ-операции с заданным типом span.
func TestStartSpan(t *testing.T) {
	// Arrange
	exporter := newTraceExporter(t)

	// Act
	_, span := startSpan(
		context.Background(),
		"rabbitmq.publish",
		"publish",
		"goph-profile.avatar",
		"avatar.process",
		oteltrace.SpanKindProducer,
	)
	span.end()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "rabbitmq.publish", spans[0].Name)
	assert.Equal(t, oteltrace.SpanKindProducer, spans[0].SpanKind)
	assert.Equal(t, traceInstrumentationName, spans[0].InstrumentationScope.Name)
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.system", "rabbitmq"))
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.operation.name", "publish"))
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.destination.name", "goph-profile.avatar"))
	assert.Contains(t, spans[0].Attributes, attribute.String("messaging.rabbitmq.routing_key", "avatar.process"))
}

// TestRabbitMQSpan_RecordError проверяет запись ошибки в span операции RabbitMQ.
func TestRabbitMQSpan_RecordError(t *testing.T) {
	// Arrange
	exporter := newTraceExporter(t)
	_, span := startConsumeSpanWithExtractedTraceContext(context.Background(), amqp.Delivery{}, "rabbitmq.consume")

	// Act
	span.recordError(assert.AnError)
	span.end()

	// Assert
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
	assert.Equal(t, assert.AnError.Error(), spans[0].Status.Description)
	require.Len(t, spans[0].Events, 1)
	assert.Equal(t, "exception", spans[0].Events[0].Name)
}

// TestAMQPTableCarrier_PropagatesTraceContext проверяет передачу контекста трассировки через заголовки AMQP.
func TestAMQPTableCarrier_PropagatesTraceContext(t *testing.T) {
	// Arrange
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTextMapPropagator(previousPropagator)
	})

	traceID := oteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := oteltrace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	parentContext := oteltrace.ContextWithSpanContext(
		context.Background(),
		oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: oteltrace.FlagsSampled,
		}),
	)
	headers := amqp.Table{}

	// Act
	otel.GetTextMapPropagator().Inject(parentContext, amqpTableCarrier(headers))
	childContext := otel.GetTextMapPropagator().Extract(context.Background(), amqpTableCarrier(headers))

	// Assert
	spanContext := oteltrace.SpanContextFromContext(childContext)
	assert.Equal(t, traceID, spanContext.TraceID())
	assert.Equal(t, spanID, spanContext.SpanID())
	assert.NotEmpty(t, amqpTableCarrier(headers).Keys())
}

// TestAMQPTableCarrier_GetBytes проверяет чтение значения заголовка, если RabbitMQ вернул его как []byte.
func TestAMQPTableCarrier_GetBytes(t *testing.T) {
	// Arrange
	carrier := amqpTableCarrier(amqp.Table{"traceparent": []byte("value")})

	// Act
	value := carrier.Get("traceparent")

	// Assert
	assert.Equal(t, "value", value)
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
