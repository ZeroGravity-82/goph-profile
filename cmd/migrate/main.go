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

	migrateApp "github.com/ZeroGravity-82/goph-profile/internal/app/migrate"
	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/observability"
)

const (
	serviceName              = "goph-profile-migrate"
	telemetryShutdownTimeout = 5 * time.Second
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadMigrate()
	if err != nil {
		if errors.Is(err, config.ErrHelp) {
			// Пользователь запросил справку по флагам командной строки; это штатное завершение.
			return
		}

		// Логгер еще не сконфигурирован.
		log.Fatalf("config error: %v", err)
	}

	logger, shutdownLogger, err := newLogger(ctx, cfg.Logging)
	if err != nil {
		// Логгер еще не сконфигурирован.
		log.Fatalf("logger config error: %v", err)
	}

	exitCode := 0
	if err := run(cfg, logger); err != nil {
		logger.ErrorContext(ctx, "migration terminated with error", slog.Any("err", err))
		exitCode = 1
	}

	if shutdownLogger != nil {
		if err := shutdownTelemetryProvider(shutdownLogger); err != nil {
			logger.ErrorContext(ctx, "failed to shutdown logger provider", slog.Any("err", err))
		}
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func newLogger(
	ctx context.Context,
	cfg config.Logging,
) (*slog.Logger, func(context.Context) error, error) {
	if !otelLoggingEnabled() {
		logger, err := observability.NewStdoutLogger(cfg)
		return logger, nil, err
	}
	return observability.NewLogger(ctx, cfg, serviceName)
}

func otelLoggingEnabled() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""
}

func run(cfg config.MigrateConfig, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	migrator, err := migrateApp.New(cfg.DatabaseURI, logger)
	if err != nil {
		return fmt.Errorf("migrator init error: %w", err)
	}
	defer func() {
		if err := migrator.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close migrator", slog.Any("err", err))
		}
	}()

	return migrator.Run(ctx)
}

func shutdownTelemetryProvider(shutdown func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), telemetryShutdownTimeout)
	defer cancel()

	return shutdown(ctx)
}
