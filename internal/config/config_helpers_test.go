package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setArgs(t *testing.T, args ...string) {
	t.Helper()

	oldArgs := os.Args
	os.Args = args
	t.Cleanup(func() {
		os.Args = oldArgs
	})
}

func setRequiredFileStorageEnv(t *testing.T) {
	t.Helper()

	t.Setenv("GOPH_PROFILE_FILE_STORAGE_ENDPOINT", "localhost:9000")
	t.Setenv("GOPH_PROFILE_FILE_STORAGE_ACCESS_KEY", "access")
	t.Setenv("GOPH_PROFILE_FILE_STORAGE_SECRET_KEY", "secret")
	t.Setenv("GOPH_PROFILE_FILE_STORAGE_BUCKET", "goph-profile")
}

func setRequiredQueueEnv(t *testing.T) {
	t.Helper()

	t.Setenv("GOPH_PROFILE_QUEUE_URL", "amqp://user:pass@localhost:5672/")
	t.Setenv("GOPH_PROFILE_QUEUE_EXCHANGE", "goph-profile.avatar")
	t.Setenv("GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_QUEUE", "goph-profile.avatar-processing")
	t.Setenv("GOPH_PROFILE_QUEUE_AVATAR_DELETION_QUEUE", "goph-profile.avatar-deletion")
	t.Setenv("GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_ROUTING_KEY", "avatar.process")
	t.Setenv("GOPH_PROFILE_QUEUE_AVATAR_DELETION_ROUTING_KEY", "avatar.delete")
}

func unsetConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"GOPH_PROFILE_HTTP_ADDRESS",
		"GOPH_PROFILE_HEALTH_ADDRESS",
		"GOPH_PROFILE_TLS_CERT",
		"GOPH_PROFILE_TLS_KEY",
		"GOPH_PROFILE_DATABASE_URI",
		"GOPH_PROFILE_FILE_STORAGE_ENDPOINT",
		"GOPH_PROFILE_FILE_STORAGE_ACCESS_KEY",
		"GOPH_PROFILE_FILE_STORAGE_SECRET_KEY",
		"GOPH_PROFILE_FILE_STORAGE_BUCKET",
		"GOPH_PROFILE_FILE_STORAGE_USE_SSL",
		"GOPH_PROFILE_QUEUE_URL",
		"GOPH_PROFILE_QUEUE_EXCHANGE",
		"GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_QUEUE",
		"GOPH_PROFILE_QUEUE_AVATAR_DELETION_QUEUE",
		"GOPH_PROFILE_QUEUE_AVATAR_PROCESSING_ROUTING_KEY",
		"GOPH_PROFILE_QUEUE_AVATAR_DELETION_ROUTING_KEY",
		"GOPH_PROFILE_LOGGING_FORMAT",
		"GOPH_PROFILE_LOGGING_LEVEL",
		"GOPH_PROFILE_LOGGING_ADD_SOURCE",
	} {
		unsetEnv(t, key)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	oldValue, existed := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if existed {
			require.NoError(t, os.Setenv(key, oldValue))
			return
		}
		require.NoError(t, os.Unsetenv(key))
	})
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)
	return path
}
