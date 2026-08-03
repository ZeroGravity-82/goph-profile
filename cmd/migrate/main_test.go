package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

// Test_run_ReturnsMigrationError проверяет, что ошибка миграции возвращается вызывающему коду.
func Test_run_ReturnsMigrationError(t *testing.T) {
	// Arrange
	cfg := config.MigrateConfig{DatabaseURI: "://bad-database-uri"}
	logger := logging.NopLogger()

	// Act
	err := run(cfg, logger)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to migrate database")
	assert.Contains(t, err.Error(), "failed to connect to the database")
}

// Test_newLogger_EndpointSet_ReturnsOTELShutdown проверяет включение отправки логов через OpenTelemetry при заданном
// адресе коллектора.
func Test_newLogger_EndpointSet_ReturnsOTELShutdown(t *testing.T) {
	// Arrange
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4317")
	cfg := config.Logging{Format: "json", Level: "info"}

	// Act
	logger, shutdown, err := newLogger(context.Background(), cfg)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, logger)
	require.NotNil(t, shutdown)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, shutdown(ctx))
	})
}

// Test_newLogger_EndpointMissing_ReturnsNoShutdown проверяет запись логов только в стандартный вывод, если адрес
// коллектора не задан.
func Test_newLogger_EndpointMissing_ReturnsNoShutdown(t *testing.T) {
	// Arrange
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	cfg := config.Logging{Format: "json", Level: "info"}

	// Act
	logger, shutdown, err := newLogger(context.Background(), cfg)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Nil(t, shutdown)
}
