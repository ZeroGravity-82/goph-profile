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

	workerApp "github.com/ZeroGravity-82/goph-profile/internal/app/worker"
	"github.com/ZeroGravity-82/goph-profile/internal/config"
)

func main() {
	cfg, err := config.LoadWorker()
	if err != nil {
		if errors.Is(err, config.ErrHelp) {
			// Пользователь запросил справку по флагам командной строки; это штатное завершение.
			return
		}

		// Логгер еще не сконфигурирован.
		log.Fatalf("config error: %v", err)
	}

	logger, err := newLogger(cfg.Logging)
	if err != nil {
		// Логгер еще не сконфигурирован.
		log.Fatalf("logger config error: %v", err)
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("worker terminated with error", slog.Any("err", err))
		os.Exit(1)
	}
	logger.Info("worker stopped (graceful)")
}

func run(cfg config.WorkerConfig, logger *slog.Logger) error {
	application, err := workerApp.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("app init error: %w", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			logger.Error("failed to close application", slog.Any("err", err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return application.Run(ctx)
}
