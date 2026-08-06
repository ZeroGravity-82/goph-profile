package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
)

// TestLevelHandler_FiltersByLevel проверяет, что levelHandler не пропускает записи ниже настроенного уровня.
func TestLevelHandler_FiltersByLevel(t *testing.T) {
	// Arrange
	tests := []struct {
		name        string
		level       slog.Level
		message     string
		wantVisible bool
	}{
		{name: "below configured level", level: slog.LevelInfo, message: "hidden", wantVisible: false},
		{name: "at configured level", level: slog.LevelWarn, message: "visible", wantVisible: true},
		{name: "above configured level", level: slog.LevelError, message: "visible", wantVisible: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logBuffer := &bytes.Buffer{}
			logger := slog.New(levelHandler{
				handler: slog.NewTextHandler(logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}),
				level:   slog.LevelWarn,
			})

			// Act
			logger.Log(context.Background(), tt.level, tt.message)

			// Assert
			logOutput := logBuffer.String()
			if tt.wantVisible {
				assert.Contains(t, logOutput, tt.message)
				return
			}
			assert.NotContains(t, logOutput, tt.message)
		})
	}
}

// TestLevelHandler_WithAttrsAndGroup проверяет сохранение атрибутов и группы при оборачивании slog.Handler.
func TestLevelHandler_WithAttrsAndGroup(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	handler := levelHandler{
		handler: slog.NewTextHandler(logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}),
		level:   slog.LevelDebug,
	}
	logger := slog.New(handler.WithAttrs([]slog.Attr{slog.String("component", "test")}).WithGroup("payload"))

	// Act
	logger.Info("grouped", slog.String("id", "42"))

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "grouped")
	assert.Contains(t, logOutput, "component=test")
	assert.Contains(t, logOutput, "payload.id=42")
}

// TestFanoutHandler_WritesToAllHandlers проверяет, что fanoutHandler отправляет запись во все нижележащие хендлеры.
func TestFanoutHandler_WritesToAllHandlers(t *testing.T) {
	// Arrange
	firstLogBuffer := &bytes.Buffer{}
	secondLogBuffer := &bytes.Buffer{}
	logger := slog.New(fanoutHandler{handlers: []slog.Handler{
		slog.NewTextHandler(firstLogBuffer, nil),
		slog.NewTextHandler(secondLogBuffer, nil),
	}})

	// Act
	logger.Info("visible")

	// Assert
	assert.Contains(t, firstLogBuffer.String(), "visible")
	assert.Contains(t, secondLogBuffer.String(), "visible")
}

// Test_newStdoutLogHandler_EnablesDebugLevel проверяет, что stdoutHandler не отсекает записи с уровнем debug.
func Test_newStdoutLogHandler_EnablesDebugLevel(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{name: "text", format: "text"},
		{name: "json", format: "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			logBuffer := &bytes.Buffer{}

			// Act
			handler, err := newStdoutLogHandler(logBuffer, config.Logging{Format: tt.format})

			// Assert
			require.NoError(t, err)
			assert.True(t, handler.Enabled(context.Background(), slog.LevelDebug))
		})
	}
}

// Test_parseLogLevel проверяет поддерживаемые уровни логирования.
func Test_parseLogLevel(t *testing.T) {
	// Arrange
	tests := []struct {
		name    string
		value   string
		want    slog.Level
		wantErr string
	}{
		{name: "debug", value: "debug", want: slog.LevelDebug},
		{name: "info upper case", value: "INFO", want: slog.LevelInfo},
		{name: "warn", value: "warn", want: slog.LevelWarn},
		{name: "error", value: "error", want: slog.LevelError},
		{name: "unknown", value: "trace", wantErr: `unsupported log level: "trace"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got, err := parseLogLevel(tt.value)

			// Assert
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestStdoutLogHandler_WritesText проверяет настройку текстового формата логов для stdout.
func TestStdoutLogHandler_WritesText(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	handler, err := newStdoutLogHandler(logBuffer, config.Logging{Format: "text"})
	require.NoError(t, err)
	logger := slog.New(handler)

	// Act
	logger.Info("visible", slog.String("component", "test"))

	// Assert
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "msg=visible")
	assert.Contains(t, logOutput, "component=test")
}

// TestStdoutLogHandler_WritesJSON проверяет настройку JSON-формата логов для stdout.
func TestStdoutLogHandler_WritesJSON(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	handler, err := newStdoutLogHandler(logBuffer, config.Logging{Format: "json"})
	require.NoError(t, err)
	logger := slog.New(handler)

	// Act
	logger.Info("visible", slog.String("component", "test"))

	// Assert
	var record map[string]any
	require.NoError(t, json.Unmarshal(logBuffer.Bytes(), &record))
	assert.Equal(t, "visible", record["msg"])
	assert.Equal(t, "test", record["component"])
}

// Test_newStdoutLogHandler_RejectsBadFormat проверяет ошибку настройки формата логов.
func Test_newStdoutLogHandler_RejectsBadFormat(t *testing.T) {
	// Arrange
	logBuffer := &bytes.Buffer{}
	cfg := config.Logging{Format: "yaml"}

	// Act
	handler, err := newStdoutLogHandler(logBuffer, cfg)

	// Assert
	require.Error(t, err)
	assert.Equal(t, `unsupported log format: "yaml"`, err.Error())
	assert.Nil(t, handler)
}

// TestNewStdoutLogger_RejectsBadLevel проверяет ошибку настройки уровня stdout-логгера.
func TestNewStdoutLogger_RejectsBadLevel(t *testing.T) {
	// Arrange
	cfg := config.Logging{Level: "trace", Format: "json"}

	// Act
	logger, err := NewStdoutLogger(cfg)

	// Assert
	require.Error(t, err)
	assert.Equal(t, `unsupported log level: "trace"`, err.Error())
	assert.Nil(t, logger)
}

// TestNewStdoutLogger_RejectsBadFormat проверяет ошибку настройки формата stdout-логгера.
func TestNewStdoutLogger_RejectsBadFormat(t *testing.T) {
	// Arrange
	cfg := config.Logging{Level: "info", Format: "yaml"}

	// Act
	logger, err := NewStdoutLogger(cfg)

	// Assert
	require.Error(t, err)
	assert.Equal(t, `unsupported log format: "yaml"`, err.Error())
	assert.Nil(t, logger)
}

// TestNewLogger_RejectsBadLevel проверяет ошибку настройки уровня логирования.
func TestNewLogger_RejectsBadLevel(t *testing.T) {
	// Arrange
	cfg := config.Logging{Level: "trace", Format: "json"}

	// Act
	logger, shutdown, err := NewLogger(context.Background(), cfg, "test")

	// Assert
	require.Error(t, err)
	assert.Equal(t, `unsupported log level: "trace"`, err.Error())
	assert.Nil(t, logger)
	assert.Nil(t, shutdown)
}

// TestNewLogger_RejectsBadFormat проверяет ошибку настройки формата логирования.
func TestNewLogger_RejectsBadFormat(t *testing.T) {
	// Arrange
	cfg := config.Logging{Level: "info", Format: "yaml"}

	// Act
	logger, shutdown, err := NewLogger(context.Background(), cfg, "test")

	// Assert
	require.Error(t, err)
	assert.Equal(t, `unsupported log format: "yaml"`, err.Error())
	assert.Nil(t, logger)
	assert.Nil(t, shutdown)
}
