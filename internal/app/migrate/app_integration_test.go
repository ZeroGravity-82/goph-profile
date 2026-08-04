//go:build integration

package migrate

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigrator_Run_Integration применяет встроенные миграции к реальной PostgreSQL.
func TestMigrator_Run_Integration(t *testing.T) {
	// Arrange
	databaseURI := os.Getenv("TEST_DATABASE_URI")
	if databaseURI == "" {
		t.Skip("TEST_DATABASE_URI is not set")
	}
	migrator, err := New(databaseURI, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, migrator.Close()) })

	// Act
	err = migrator.Run(context.Background())

	// Assert
	require.NoError(t, err)
}
