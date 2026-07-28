package postgres

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const traceInstrumentationName = "goph-profile.postgres"

// startSpan создает span трассировки для операции с PostgreSQL и добавляет общие атрибуты базы данных.
func startSpan(
	ctx context.Context,
	name string,
	operation string,
	collection string,
) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.operation.name", operation),
	}
	if collection != "" {
		attrs = append(attrs, attribute.String("db.collection.name", collection))
	}

	return otel.Tracer(traceInstrumentationName).Start(
		ctx,
		name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

// recordSpanError записывает ошибку в span PostgreSQL и помечает span как неуспешный.
func recordSpanError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
