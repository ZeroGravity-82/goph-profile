package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

// Test_run_ReturnsAppInitError проверяет, что ошибка инициализации приложения возвращается вызывающему коду.
func Test_run_ReturnsAppInitError(t *testing.T) {
	// Arrange
	cfg := config.ServerConfig{DatabaseURI: "://bad-database-uri"}
	logger := logging.NopLogger()

	// Act
	err := run(cfg, logger)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app init error")
}
