//go:build integration

package migrate

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
)

// TestRun_Integration применяет встроенные миграции к реальной PostgreSQL.
func TestRun_Integration(t *testing.T) {
	// Arrange
	databaseURI := os.Getenv("TEST_DATABASE_URI")
	if databaseURI == "" {
		t.Skip("TEST_DATABASE_URI is not set")
	}

	// Act
	err := Run(context.Background(), config.MigrateConfig{DatabaseURI: databaseURI}, nil)

	// Assert
	require.NoError(t, err)
}
