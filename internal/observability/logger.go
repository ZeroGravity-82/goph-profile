package observability

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	globalLog "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
)

// levelHandler добавляет фильтрацию по уровню логирования для нижележащего хендлера.
// Без этой обертки cfg.Level не применялся бы единообразно к stdout и OpenTelemetry-хендлерам.
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

// fanoutHandler передает одну запись логирования нескольким нижележащим хендлерам.
type fanoutHandler struct {
	handlers []slog.Handler
}

func (h fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var err error
	for _, handler := range h.handlers {
		// У каждого нижележащего хендлера может быть свой порог уровня логирования.
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		err = errors.Join(err, handler.Handle(ctx, record.Clone()))
	}
	return err
}

func (h fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithAttrs(attrs))
	}
	return fanoutHandler{handlers: handlers}
}

func (h fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		handlers = append(handlers, handler.WithGroup(name))
	}
	return fanoutHandler{handlers: handlers}
}

// NewLogger создает slog-логгер для записи в stdout и отправки записей в OTEL Collector.
func NewLogger(
	ctx context.Context,
	cfg config.Logging,
	serviceName string,
) (*slog.Logger, func(context.Context) error, error) {
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	stdoutHandler, err := newStdoutLogHandler(os.Stdout, cfg)
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

	otelHandler := otelslog.NewHandler(
		logInstrumentationName,
		otelslog.WithLoggerProvider(loggerProvider),
		otelslog.WithSource(cfg.AddSource),
	)
	logger := slog.New(levelHandler{
		handler: fanoutHandler{handlers: []slog.Handler{
			stdoutHandler,
			otelHandler,
		}},
		level: level,
	})

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

// newStdoutLogHandler создает stdout-хендлер в формате, выбранном в конфигурации.
func newStdoutLogHandler(w io.Writer, cfg config.Logging) (slog.Handler, error) {
	options := &slog.HandlerOptions{AddSource: cfg.AddSource, Level: slog.LevelDebug}

	switch strings.ToLower(cfg.Format) {
	case "json":
		return slog.NewJSONHandler(w, options), nil
	case "text":
		return slog.NewTextHandler(w, options), nil
	default:
		return nil, fmt.Errorf("unsupported log format: %q", cfg.Format)
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
