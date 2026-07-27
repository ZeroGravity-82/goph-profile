package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// AvatarUseCaseMetrics содержит метрики пользовательских сценариев.
//
// uploads считает запросы загрузки аватарок с разбивкой по статусу выполнения и коду причины.
//
// uploadSizeBytes хранит распределение размеров загружаемых файлов.
//
// useCaseActions считает вызовы пользовательских сценариев аватарок.
//
// useCaseFailures считает ошибки пользовательских сценариев аватарок.
type AvatarUseCaseMetrics struct {
	uploads         metric.Int64Counter
	uploadSizeBytes metric.Int64Histogram
	useCaseActions  metric.Int64Counter
	useCaseFailures metric.Int64Counter
}

// NewAvatarUseCaseMetrics создает метрики пользовательских сценариев аватарок.
func NewAvatarUseCaseMetrics() (*AvatarUseCaseMetrics, error) {
	meter := otel.Meter(metricInstrumentationName)

	uploads, err := meter.Int64Counter(
		"avatars_uploads_total",
		metric.WithDescription("Total number of avatar upload requests."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar uploads metric: %w", err)
	}
	uploadSize, err := meter.Int64Histogram(
		"avatars_upload_size_bytes",
		metric.WithDescription("Uploaded avatar file size."),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar upload size metric: %w", err)
	}
	useCaseActions, err := meter.Int64Counter(
		"avatar_api_actions_total",
		metric.WithDescription("Total number of avatar use case actions."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar use case actions metric: %w", err)
	}
	useCaseFailures, err := meter.Int64Counter(
		"avatar_api_failures_total",
		metric.WithDescription("Total number of avatar use case failures."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar use case failures metric: %w", err)
	}

	return &AvatarUseCaseMetrics{
		uploads:         uploads,
		uploadSizeBytes: uploadSize,
		useCaseActions:  useCaseActions,
		useCaseFailures: useCaseFailures,
	}, nil
}

// RecordAvatarUpload записывает результат загрузки аватарки.
func (m *AvatarUseCaseMetrics) RecordAvatarUpload(
	ctx context.Context,
	status string,
	reason string,
	mimeType string,
	sizeBytes int64,
) {
	attrs := metric.WithAttributes(
		attribute.String("status", status),
		attribute.String("reason", reason),
		attribute.String("mime_type", mimeType),
	)

	m.uploads.Add(ctx, 1, attrs)
	if sizeBytes > 0 {
		m.uploadSizeBytes.Record(ctx, sizeBytes, attrs)
	}
}

// RecordAvatarUseCaseAction записывает результат операции пользовательского сценария аватарок.
func (m *AvatarUseCaseMetrics) RecordAvatarUseCaseAction(
	ctx context.Context,
	action string,
	status string,
	reason string,
) {
	attrs := metric.WithAttributes(
		attribute.String("action", action),
		attribute.String("status", status),
		attribute.String("reason", reason),
	)

	m.useCaseActions.Add(ctx, 1, attrs)
	if status == "error" {
		m.useCaseFailures.Add(ctx, 1, attrs)
	}
}
