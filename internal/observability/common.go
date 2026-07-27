package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

const (
	logInstrumentationName    = "goph-profile.logs"
	metricInstrumentationName = "goph-profile.metrics"
)

// newResource задает общие OTEL-метаданные (атрибуты) сервиса для логов, метрик и трассировки.
func newResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
		resource.WithAttributes(attribute.String("service.name", serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTEL resource: %w", err)
	}

	return res, nil
}
