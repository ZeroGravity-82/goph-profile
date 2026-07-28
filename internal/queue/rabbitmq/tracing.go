package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const traceInstrumentationName = "goph-profile.rabbitmq"

// rabbitMQSpan скрывает детали OpenTelemetry span для операций RabbitMQ.
type rabbitMQSpan struct {
	span trace.Span
}

// setMessageID добавляет ID сообщения в span.
func (s rabbitMQSpan) setMessageID(messageID string) {
	s.span.SetAttributes(attribute.String("messaging.message.id", messageID))
}

// setAvatarID добавляет ID аватарки в span после успешного парсинга сообщения.
func (s rabbitMQSpan) setAvatarID(avatarID string) {
	s.span.SetAttributes(attribute.String("avatar.id", avatarID))
}

// recordError записывает ошибку в span и помечает span как неуспешный.
func (s rabbitMQSpan) recordError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// end завершает span операции RabbitMQ.
func (s rabbitMQSpan) end() {
	s.span.End()
}

// startConsumeSpanWithExtractedTraceContext извлекает родительский контекст трассировки из заголовков AMQP, начинает
// span обработки сообщения и добавляет ID сообщения RabbitMQ.
func startConsumeSpanWithExtractedTraceContext(
	ctx context.Context,
	delivery amqp.Delivery,
	name string,
) (context.Context, rabbitMQSpan) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, amqpTableCarrier(delivery.Headers))
	ctx, span := startSpan(ctx, name, "consume", delivery.Exchange, delivery.RoutingKey, trace.SpanKindConsumer)
	span.setMessageID(delivery.MessageId)

	return ctx, span
}

// startSpan создает span трассировки для операции с RabbitMQ и добавляет общие атрибуты RabbitMQ-операции.
func startSpan(
	ctx context.Context,
	name string,
	operation string,
	exchange string,
	routingKey string,
	spanKind trace.SpanKind,
) (context.Context, rabbitMQSpan) {
	ctx, span := otel.Tracer(traceInstrumentationName).Start(
		ctx,
		name,
		trace.WithSpanKind(spanKind),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.operation.name", operation),
			attribute.String("messaging.destination.name", exchange),
			attribute.String("messaging.rabbitmq.routing_key", routingKey),
		),
	)

	return ctx, rabbitMQSpan{span: span}
}

// amqpTableCarrier адаптирует заголовки AMQP к интерфейсу OpenTelemetry для передачи контекста трассировки через
// RabbitMQ.
type amqpTableCarrier amqp.Table

// Get возвращает значение заголовка AMQP по имени для извлечения контекста трассировки.
func (c amqpTableCarrier) Get(key string) string {
	value, ok := amqp.Table(c)[key]
	if !ok {
		return ""
	}
	switch typedValue := value.(type) {
	case string:
		return typedValue
	case []byte:
		return string(typedValue)
	default:
		return ""
	}
}

// Set записывает значение контекста трассировки в заголовки AMQP перед публикацией сообщения.
func (c amqpTableCarrier) Set(key string, value string) {
	amqp.Table(c)[key] = value
}

// Keys возвращает имена заголовков AMQP, доступные пропагатору OpenTelemetry.
func (c amqpTableCarrier) Keys() []string {
	keys := make([]string, 0, len(amqp.Table(c)))
	for key := range amqp.Table(c) {
		keys = append(keys, key)
	}
	return keys
}
