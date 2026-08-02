package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadMigrate_HelpReturnsErrHelp проверяет, что запрос справки возвращает специальную ошибку.
func TestLoadMigrate_HelpReturnsErrHelp(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	setArgs(t, "migrate", "-h")

	// Act
	_, err := LoadMigrate()

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHelp))
}

// TestLoadMigrate_RequiresDatabaseURI проверяет обязательность строки подключения к БД.
func TestLoadMigrate_RequiresDatabaseURI(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	setArgs(t, "migrate")

	// Act
	_, err := LoadMigrate()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database URI is required")
}

// TestLoadMigrate_LoadsDefaults проверяет значения по умолчанию.
func TestLoadMigrate_LoadsDefaults(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
	setArgs(t, "migrate")

	// Act
	cfg, err := LoadMigrate()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseURI)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.False(t, cfg.Logging.AddSource)
}

// TestLoadMigrate_Priority проверяет приоритет источников конфигурации мигратора.
func TestLoadMigrate_Priority(t *testing.T) {
	// Arrange
	configPath := writeTempConfig(t, `
database_uri: postgres://file-db
logging:
  format: json
  level: warn
  add_source: false
`)
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://env-db")
	t.Setenv("GOPH_PROFILE_LOGGING_LEVEL", "error")
	setArgs(t,
		"migrate",
		"--config", configPath,
		"--database-uri", "postgres://flag-db",
		"--logging.format", "text",
		"--logging.add-source",
	)

	// Act
	cfg, err := LoadMigrate()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "postgres://flag-db", cfg.DatabaseURI)
	assert.Equal(t, "text", cfg.Logging.Format)
	assert.Equal(t, "error", cfg.Logging.Level)
	assert.True(t, cfg.Logging.AddSource)
}
