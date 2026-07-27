package observability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	globalLog "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
)

// levelHandler добавляет фильтрацию по уровню логирования к хендлеру OpenTelemetry.
// Без этой обертки otelslog примет записи ниже настроенного уровня логирования, и они уйдут в OTEL Collector.
type levelHandler struct {
	handler slog.Handler
	level   slog.Level
}

func (h levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.handler.Enabled(ctx, level)
}

func (h levelHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.handler.Handle(ctx, record)
}

func (h levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelHandler{
		handler: h.handler.WithAttrs(attrs),
		level:   h.level,
	}
}

func (h levelHandler) WithGroup(name string) slog.Handler {
	return levelHandler{
		handler: h.handler.WithGroup(name),
		level:   h.level,
	}
}

// NewLogger создает slog-логгер для отправки записей в OTEL Collector.
func NewLogger(
	ctx context.Context,
	cfg config.Logging,
	serviceName string,
) (*slog.Logger, func(context.Context) error, error) {
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	loggerProvider, err := newLoggerProvider(ctx, serviceName)
	if err != nil {
		return nil, nil, err
	}
	// Глобальный провайдер нужен коду, который пишет логи через OpenTelemetry Logs API напрямую.
	// Например, это может делать код сторонней библиотеки.
	globalLog.SetLoggerProvider(loggerProvider)

	logger := slog.New(levelHandler{
		handler: otelslog.NewHandler(
			logInstrumentationName,
			otelslog.WithLoggerProvider(loggerProvider),
			otelslog.WithSource(cfg.AddSource),
		),
		level: level,
	})
	slog.SetDefault(logger)

	return logger, loggerProvider.Shutdown, nil
}

// parseLogLevel переводит строковый уровень логирования из конфигурации в slog.Level.
func parseLogLevel(v string) (slog.Level, error) {
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %q", v)
	}
}

// newLoggerProvider создает OTEL LoggerProvider для отправки логов в OTEL Collector через OTLP.
func newLoggerProvider(ctx context.Context, serviceName string) (*sdklog.LoggerProvider, error) {
	exporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	res, err := newResource(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	), nil
}
