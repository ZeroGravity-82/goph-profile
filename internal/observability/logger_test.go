package observability

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
)

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
