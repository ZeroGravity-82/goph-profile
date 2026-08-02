package main

import (
	"testing"

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
