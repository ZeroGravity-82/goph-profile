package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	workerApp "github.com/ZeroGravity-82/goph-profile/internal/app/worker"
	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/observability"
)

const (
	serviceName              = "goph-profile-worker"
	telemetryShutdownTimeout = 5 * time.Second
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadWorker()
	if err != nil {
		if errors.Is(err, config.ErrHelp) {
			// Пользователь запросил справку по флагам командной строки; это штатное завершение.
			return
		}

		// Логгер еще не сконфигурирован.
		log.Fatalf("config error: %v", err)
	}

	logger, shutdownLogger, err := observability.NewLogger(ctx, cfg.Logging, serviceName)
	if err != nil {
		// Логгер еще не сконфигурирован.
		log.Fatalf("logger config error: %v", err)
	}
	shutdownMeter, err := observability.SetupGlobalMeterProvider(ctx, serviceName)
	if err != nil {
		_ = shutdownTelemetryProvider(shutdownLogger)
		log.Fatalf("meter config error: %v", err)
	}
	shutdownTracer, err := observability.SetupGlobalTracerProvider(ctx, serviceName)
	if err != nil {
		_ = shutdownTelemetryProvider(shutdownMeter)
		_ = shutdownTelemetryProvider(shutdownLogger)
		log.Fatalf("tracer config error: %v", err)
	}

	exitCode := 0
	if err := run(cfg, logger); err != nil {
		logger.ErrorContext(ctx, "worker terminated with error", slog.Any("err", err))
		exitCode = 1
	} else {
		logger.InfoContext(ctx, "worker stopped (graceful)")
	}

	if err := shutdownTelemetryProvider(shutdownTracer); err != nil {
		logger.ErrorContext(ctx, "failed to shutdown tracer provider", slog.Any("err", err))
	}
	if err := shutdownTelemetryProvider(shutdownMeter); err != nil {
		logger.ErrorContext(ctx, "failed to shutdown meter provider", slog.Any("err", err))
	}
	if err := shutdownTelemetryProvider(shutdownLogger); err != nil {
		logger.ErrorContext(ctx, "failed to shutdown logger provider", slog.Any("err", err))
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func run(cfg config.WorkerConfig, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := workerApp.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("app init error: %w", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close application", slog.Any("err", err))
		}
	}()

	return application.Run(ctx)
}

func shutdownTelemetryProvider(shutdown func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), telemetryShutdownTimeout)
	defer cancel()

	return shutdown(ctx)
}
