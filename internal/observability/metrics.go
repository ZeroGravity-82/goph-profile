package observability

import (
	"context"
	"fmt"

	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// SetupGlobalMeterProvider настраивает глобальный OTEL MeterProvider и запускает метрики рантайма Go.
func SetupGlobalMeterProvider(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	res, err := newResource(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)

	if err = otelruntime.Start(otelruntime.WithMeterProvider(meterProvider)); err != nil {
		_ = meterProvider.Shutdown(ctx)
		return nil, fmt.Errorf("failed to start Go runtime metrics: %w", err)
	}

	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}
