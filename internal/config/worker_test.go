package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadWorker_HelpReturnsErrHelp проверяет, что запрос справки возвращает специальную ошибку.
func TestLoadWorker_HelpReturnsErrHelp(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	setArgs(t, "worker", "-h")

	// Act
	_, err := LoadWorker()

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHelp))
}

// TestLoadWorker_RequiresCommonFields проверяет обязательность общих настроек воркера.
func TestLoadWorker_RequiresCommonFields(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	setRequiredFileStorageEnv(t)
	setRequiredQueueEnv(t)
	setArgs(t, "worker")

	// Act
	_, err := LoadWorker()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database URI is required")
}

// TestLoadWorker_RejectsInvalidHealthServerAddr проверяет ошибку при некорректном адресе сервера проверок состояния.
func TestLoadWorker_RejectsInvalidHealthServerAddr(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
	setRequiredFileStorageEnv(t)
	setRequiredQueueEnv(t)
	setArgs(t, "worker", "--health-address", "http://localhost:3203")

	// Act
	_, err := LoadWorker()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server address must be in the format host:port")
}

// TestLoadWorker_LoadsDefaultHealthServerAddr проверяет адрес сервера проверок состояния по умолчанию.
func TestLoadWorker_LoadsDefaultHealthServerAddr(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
	setRequiredFileStorageEnv(t)
	setRequiredQueueEnv(t)
	setArgs(t, "worker")

	// Act
	cfg, err := LoadWorker()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "localhost:3203", cfg.HealthServerAddr)
}

// TestLoadWorker_Priority проверяет приоритет источников конфигурации воркера.
func TestLoadWorker_Priority(t *testing.T) {
	// Arrange
	configPath := writeTempConfig(t, `
health_address: localhost:3204
database_uri: postgres://file-db
file_storage:
  endpoint: localhost:9000
  access_key: file-access
  secret_key: file-secret
  bucket: file-bucket
  use_ssl: true
queue:
  url: amqp://file-user:file-pass@localhost:5672/
  exchange: file-exchange
  avatar_processing_queue: file-process-queue
  avatar_deletion_queue: file-delete-queue
  avatar_processing_routing_key: file.process
  avatar_deletion_routing_key: file.delete
logging:
  format: json
  level: warn
  add_source: false
`)
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_HEALTH_ADDRESS", "127.0.0.1:3204")
	t.Setenv("GOPH_PROFILE_FILE_STORAGE_ENDPOINT", "localhost:9002")
	t.Setenv("GOPH_PROFILE_FILE_STORAGE_BUCKET", "env-bucket")
	t.Setenv("GOPH_PROFILE_QUEUE_URL", "amqp://env-user:env-pass@localhost:5672/")
	t.Setenv("GOPH_PROFILE_QUEUE_EXCHANGE", "env-exchange")
	t.Setenv("GOPH_PROFILE_LOGGING_LEVEL", "error")
	setArgs(t,
		"worker",
		"--config", configPath,
		"--health-address", "127.0.0.1:3203",
		"--database-uri", "postgres://flag-db",
		"--file-storage.endpoint", "localhost:9001",
		"--file-storage.access-key", "flag-access",
		"--file-storage.secret-key", "flag-object-secret",
		"--file-storage.bucket", "flag-bucket",
		"--file-storage.use-ssl=false",
		"--queue.url", "amqp://flag-user:flag-pass@localhost:5672/",
		"--queue.exchange", "flag-exchange",
		"--queue.avatar-processing-queue", "flag-process-queue",
		"--queue.avatar-deletion-queue", "flag-delete-queue",
		"--queue.avatar-processing-routing-key", "flag.process",
		"--queue.avatar-deletion-routing-key", "flag.delete",
		"--logging.level", "debug",
		"--logging.add-source",
	)

	// Act
	cfg, err := LoadWorker()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:3203", cfg.HealthServerAddr)
	assert.Equal(t, "postgres://flag-db", cfg.DatabaseURI)
	assert.Equal(t, "localhost:9001", cfg.FileStorage.Endpoint)
	assert.Equal(t, "flag-access", cfg.FileStorage.AccessKey)
	assert.Equal(t, "flag-object-secret", cfg.FileStorage.SecretKey)
	assert.Equal(t, "flag-bucket", cfg.FileStorage.Bucket)
	assert.Equal(t, false, cfg.FileStorage.UseSSL)
	assert.Equal(t, "amqp://flag-user:flag-pass@localhost:5672/", cfg.Queue.URL)
	assert.Equal(t, "flag-exchange", cfg.Queue.Exchange)
	assert.Equal(t, "flag-process-queue", cfg.Queue.AvatarProcessingQueue)
	assert.Equal(t, "flag-delete-queue", cfg.Queue.AvatarDeletionQueue)
	assert.Equal(t, "flag.process", cfg.Queue.AvatarProcessingRoutingKey)
	assert.Equal(t, "flag.delete", cfg.Queue.AvatarDeletionRoutingKey)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.True(t, cfg.Logging.AddSource)
}
