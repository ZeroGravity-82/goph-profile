package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadServer_HelpReturnsErrHelp проверяет, что запрос справки возвращает специальную ошибку.
func TestLoadServer_HelpReturnsErrHelp(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	setArgs(t, "server", "-h")

	// Act
	_, err := LoadServer()

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHelp))
}

// TestLoadServer_RequiresDatabaseURI проверяет обязательность строки подключения к БД.
func TestLoadServer_RequiresDatabaseURI(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	setRequiredFileStorageEnv(t)
	setRequiredQueueEnv(t)
	setArgs(t, "server")

	// Act
	_, err := LoadServer()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database URI is required")
}

// TestLoadServer_RequiresFileStorageFields проверяет обязательность настроек файлового хранилища.
func TestLoadServer_RequiresFileStorageFields(t *testing.T) {
	tests := []struct {
		name       string
		unsetKey   string
		wantErrMsg string
	}{
		{
			name:       "endpoint",
			unsetKey:   "GOPH_PROFILE_FILE_STORAGE_ENDPOINT",
			wantErrMsg: "file storage endpoint is required",
		},
		{
			name:       "access key",
			unsetKey:   "GOPH_PROFILE_FILE_STORAGE_ACCESS_KEY",
			wantErrMsg: "file storage access key is required",
		},
		{
			name:       "secret key",
			unsetKey:   "GOPH_PROFILE_FILE_STORAGE_SECRET_KEY",
			wantErrMsg: "file storage secret key is required",
		},
		{
			name:       "bucket",
			unsetKey:   "GOPH_PROFILE_FILE_STORAGE_BUCKET",
			wantErrMsg: "file storage bucket is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			unsetConfigEnv(t)
			t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
			setRequiredFileStorageEnv(t)
			setRequiredQueueEnv(t)
			t.Setenv(tt.unsetKey, "")
			setArgs(t, "server")

			// Act
			_, err := LoadServer()

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

// TestLoadServer_RequiresQueueFields проверяет обязательность настроек очереди сообщений.
func TestLoadServer_RequiresQueueFields(t *testing.T) {
	tests := []struct {
		name       string
		unsetKey   string
		wantErrMsg string
	}{
		{
			name:       "url",
			unsetKey:   "GOPH_PROFILE_QUEUE_URL",
			wantErrMsg: "queue URL is required",
		},
		{
			name:       "exchange",
			unsetKey:   "GOPH_PROFILE_QUEUE_EXCHANGE",
			wantErrMsg: "queue exchange is required",
		},
		{
			name:       "avatar processing queue",
			unsetKey:   "GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_QUEUE",
			wantErrMsg: "queue avatar processing queue is required",
		},
		{
			name:       "avatar deletion queue",
			unsetKey:   "GOPH_PROFILE_QUEUE_AVATAR_DELETION_QUEUE",
			wantErrMsg: "queue avatar deletion queue is required",
		},
		{
			name:       "avatar processing routing key",
			unsetKey:   "GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_ROUTING_KEY",
			wantErrMsg: "queue avatar processing routing key is required",
		},
		{
			name:       "avatar deletion routing key",
			unsetKey:   "GOPH_PROFILE_QUEUE_AVATAR_DELETION_ROUTING_KEY",
			wantErrMsg: "queue avatar deletion routing key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			unsetConfigEnv(t)
			t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
			setRequiredFileStorageEnv(t)
			setRequiredQueueEnv(t)
			t.Setenv(tt.unsetKey, "")
			setArgs(t, "server")

			// Act
			_, err := LoadServer()

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

// TestLoadServer_RejectsInvalidHTTPServerAddr проверяет ошибку при некорректном формате адреса HTTP-сервера.
func TestLoadServer_RejectsInvalidHTTPServerAddr(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
	setRequiredFileStorageEnv(t)
	setRequiredQueueEnv(t)
	setArgs(t, "server", "--http-address", "http://localhost:3201")

	// Act
	_, err := LoadServer()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server address must be in the format host:port")
}

// TestLoadServer_DisablesTLSForEmptyPaths проверяет отключение TLS при пустых путях к сертификату и ключу.
func TestLoadServer_DisablesTLSForEmptyPaths(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
	t.Setenv("GOPH_PROFILE_TLS_CERT", "")
	t.Setenv("GOPH_PROFILE_TLS_KEY", "")
	setRequiredFileStorageEnv(t)
	setRequiredQueueEnv(t)
	setArgs(t, "server")

	// Act
	cfg, err := LoadServer()

	// Assert
	require.NoError(t, err)
	assert.Empty(t, cfg.TLSCertPath)
	assert.Empty(t, cfg.TLSKeyPath)
}

// TestLoadServer_RejectsIncompleteTLSConfig проверяет обязательность совместного указания сертификата и ключа.
func TestLoadServer_RejectsIncompleteTLSConfig(t *testing.T) {
	tests := []struct {
		name       string
		certPath   string
		keyPath    string
		wantErrMsg string
	}{
		{
			name:       "certificate only",
			certPath:   "certs/server.crt",
			wantErrMsg: "TLS private key path is required when TLS certificate path is provided",
		},
		{
			name:       "private key only",
			keyPath:    "certs/server.key",
			wantErrMsg: "TLS certificate path is required when TLS private key path is provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			unsetConfigEnv(t)
			t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
			t.Setenv("GOPH_PROFILE_TLS_CERT", tt.certPath)
			t.Setenv("GOPH_PROFILE_TLS_KEY", tt.keyPath)
			setRequiredFileStorageEnv(t)
			setRequiredQueueEnv(t)
			setArgs(t, "server")

			// Act
			_, err := LoadServer()

			// Assert
			require.EqualError(t, err, tt.wantErrMsg)
		})
	}
}

// TestLoadServer_LoadsDefaults проверяет значения по умолчанию.
func TestLoadServer_LoadsDefaults(t *testing.T) {
	// Arrange
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "postgres://user:pass@localhost/db")
	setRequiredFileStorageEnv(t)
	t.Setenv("GOPH_PROFILE_QUEUE_URL", "amqp://user:pass@localhost:5672/")
	setArgs(t, "server")

	// Act
	cfg, err := LoadServer()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "localhost:3201", cfg.HTTPServerAddr)
	assert.Equal(t, "certs/server.crt", cfg.TLSCertPath)
	assert.Equal(t, "certs/server.key", cfg.TLSKeyPath)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseURI)
	assert.Equal(t, "localhost:9000", cfg.FileStorage.Endpoint)
	assert.Equal(t, "access", cfg.FileStorage.AccessKey)
	assert.Equal(t, "secret", cfg.FileStorage.SecretKey)
	assert.Equal(t, "goph-profile", cfg.FileStorage.Bucket)
	assert.Equal(t, false, cfg.FileStorage.UseSSL)
	assert.Equal(t, "amqp://user:pass@localhost:5672/", cfg.Queue.URL)
	assert.Equal(t, "goph-profile.avatar", cfg.Queue.Exchange)
	assert.Equal(t, "goph-profile.avatar-processing", cfg.Queue.AvatarProcessingQueue)
	assert.Equal(t, "goph-profile.avatar-deletion", cfg.Queue.AvatarDeletionQueue)
	assert.Equal(t, "avatar.process", cfg.Queue.AvatarProcessingRoutingKey)
	assert.Equal(t, "avatar.delete", cfg.Queue.AvatarDeletionRoutingKey)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, false, cfg.Logging.AddSource)
}

// Test_validateServerAddr проверяет допустимые и недопустимые форматы сетевого адреса.
func Test_validateServerAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{name: "host and port", addr: "localhost:3201"},
		{name: "ip and port", addr: "127.0.0.1:3201"},
		{name: "empty", wantErr: true},
		{name: "without port", addr: "localhost", wantErr: true},
		{name: "without host", addr: ":3201", wantErr: true},
		{name: "with scheme", addr: "http://localhost:3201", wantErr: true},
		{name: "only port separator", addr: "localhost:", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := validateServerAddr(tt.addr)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(
					t,
					"server address must be in the format host:port (without specifying a scheme)",
					err.Error(),
				)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestLoadServer_Priority проверяет приоритет "дефолтное значение < значение из конфигурационного файла < переменная
// окружения < флаг командной строки".
func TestLoadServer_Priority(t *testing.T) {
	// Arrange
	configPath := writeTempConfig(t, `
http_address: localhost:3202
tls_cert: certs/file-server.crt
tls_key: certs/file-server.key
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
	t.Setenv("GOPH_PROFILE_HTTP_ADDRESS", "127.0.0.1:3204")
	t.Setenv("GOPH_PROFILE_TLS_CERT", "certs/env-server.crt")
	t.Setenv("GOPH_PROFILE_FILE_STORAGE_ENDPOINT", "localhost:9002")
	t.Setenv("GOPH_PROFILE_FILE_STORAGE_BUCKET", "env-bucket")
	t.Setenv("GOPH_PROFILE_QUEUE_URL", "amqp://env-user:env-pass@localhost:5672/")
	t.Setenv("GOPH_PROFILE_QUEUE_EXCHANGE", "env-exchange")
	t.Setenv("GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_QUEUE", "env-process-queue")
	t.Setenv("GOPH_PROFILE_LOGGING_LEVEL", "error")
	setArgs(t,
		"server",
		"--config", configPath,
		"--database-uri", "postgres://flag-db",
		"--http-address", "127.0.0.1:3203",
		"--tls-cert", "certs/flag-server.crt",
		"--tls-key", "certs/flag-server.key",
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
	cfg, err := LoadServer()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "postgres://flag-db", cfg.DatabaseURI)
	assert.Equal(t, "127.0.0.1:3203", cfg.HTTPServerAddr)
	assert.Equal(t, "certs/flag-server.crt", cfg.TLSCertPath)
	assert.Equal(t, "certs/flag-server.key", cfg.TLSKeyPath)
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

// TestLoadServer_EmptyEnvironmentValueOverridesLowerPrioritySources проверяет, что пустая переменная окружения не
// откатывается к нижестоящему источнику.
func TestLoadServer_EmptyEnvironmentValueOverridesLowerPrioritySources(t *testing.T) {
	// Arrange
	configPath := writeTempConfig(t, `
database_uri: postgres://file-db
file_storage:
  endpoint: localhost:9000
  access_key: access
  secret_key: object-secret
  bucket: goph-profile
queue:
  url: amqp://file-user:file-pass@localhost:5672/
`)
	unsetConfigEnv(t)
	t.Setenv("GOPH_PROFILE_DATABASE_URI", "")
	setArgs(t,
		"server",
		"--config", configPath,
	)

	// Act
	_, err := LoadServer()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database URI is required")
}
