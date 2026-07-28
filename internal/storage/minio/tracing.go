package minio

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const traceInstrumentationName = "goph-profile.minio"

// startSpan создает span трассировки для операции с MinIO и добавляет общие атрибуты объектного хранилища.
func startSpan(ctx context.Context, name string, operation string, bucket string) (context.Context, trace.Span) {
	return otel.Tracer(traceInstrumentationName).Start(
		ctx,
		name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("storage.system", "minio"),
			attribute.String("storage.operation", operation),
			attribute.String("storage.bucket.name", bucket),
		),
	)
}

// recordSpanError записывает ошибку в span MinIO и помечает span как неуспешный.
func recordSpanError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
