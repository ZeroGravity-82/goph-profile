package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// AvatarAsyncMetrics содержит метрики асинхронной обработки аватарок и удаления файлов.
//
// jobs считает задачи воркера с разбивкой по операции, статусу выполнения и коду причины.
//
// jobDuration хранит распределение длительности обработки задач.
//
// originalSizeBytes хранит распределение размеров исходных файлов.
//
// thumbnailSizeBytes хранит распределение размеров созданных миниатюр.
//
// deletedObjects считает удаленные файлы аватарок.
type AvatarAsyncMetrics struct {
	jobs               metric.Int64Counter
	jobDuration        metric.Float64Histogram
	originalSizeBytes  metric.Int64Histogram
	thumbnailSizeBytes metric.Int64Histogram
	deletedObjects     metric.Int64Counter
}

// NewAvatarAsyncMetrics создает метрики асинхронной обработки аватарок и удаления файлов.
func NewAvatarAsyncMetrics() (*AvatarAsyncMetrics, error) {
	meter := otel.Meter(metricInstrumentationName)

	jobs, err := meter.Int64Counter(
		"avatar_worker_jobs_total",
		metric.WithDescription("Total number of avatar worker jobs."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar worker jobs metric: %w", err)
	}
	jobDuration, err := meter.Float64Histogram(
		"avatar_worker_job_duration_seconds",
		metric.WithDescription("Avatar worker job duration."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar worker job duration metric: %w", err)
	}
	originalSize, err := meter.Int64Histogram(
		"avatar_worker_original_size_bytes",
		metric.WithDescription("Original avatar file size read by the worker."),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar worker original size metric: %w", err)
	}
	thumbnailSize, err := meter.Int64Histogram(
		"avatar_worker_thumbnail_size_bytes",
		metric.WithDescription("Generated avatar thumbnail size."),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar worker thumbnail size metric: %w", err)
	}
	deletedObjects, err := meter.Int64Counter(
		"avatar_worker_deleted_objects_total",
		metric.WithDescription("Total number of avatar files deleted by the worker."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar worker deleted objects metric: %w", err)
	}

	return &AvatarAsyncMetrics{
		jobs:               jobs,
		jobDuration:        jobDuration,
		originalSizeBytes:  originalSize,
		thumbnailSizeBytes: thumbnailSize,
		deletedObjects:     deletedObjects,
	}, nil
}

// RecordAvatarProcessing записывает результат обработки изображения воркером.
func (m *AvatarAsyncMetrics) RecordAvatarProcessing(
	ctx context.Context,
	status string,
	reason string,
	duration time.Duration,
	originalSizeBytes int64,
	thumb100SizeBytes int64,
	thumb300SizeBytes int64,
) {
	attrs := metric.WithAttributes(
		attribute.String("operation", "process"),
		attribute.String("status", status),
		attribute.String("reason", reason),
	)

	m.jobs.Add(ctx, 1, attrs)
	m.jobDuration.Record(ctx, duration.Seconds(), attrs)
	if originalSizeBytes > 0 {
		m.originalSizeBytes.Record(ctx, originalSizeBytes, attrs)
	}
	if thumb100SizeBytes > 0 {
		m.thumbnailSizeBytes.Record(ctx, thumb100SizeBytes, metric.WithAttributes(
			attribute.String("size", "100x100"),
			attribute.String("status", status),
			attribute.String("reason", reason),
		))
	}
	if thumb300SizeBytes > 0 {
		m.thumbnailSizeBytes.Record(ctx, thumb300SizeBytes, metric.WithAttributes(
			attribute.String("size", "300x300"),
			attribute.String("status", status),
			attribute.String("reason", reason),
		))
	}
}

// RecordAvatarDeletion записывает результат удаления файлов аватарки воркером.
func (m *AvatarAsyncMetrics) RecordAvatarDeletion(
	ctx context.Context,
	status string,
	reason string,
	duration time.Duration,
	deletedObjects int64,
) {
	attrs := metric.WithAttributes(
		attribute.String("operation", "delete"),
		attribute.String("status", status),
		attribute.String("reason", reason),
	)

	m.jobs.Add(ctx, 1, attrs)
	m.jobDuration.Record(ctx, duration.Seconds(), attrs)
	if deletedObjects > 0 {
		m.deletedObjects.Add(ctx, deletedObjects, attrs)
	}
}
