//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	migrationfiles "github.com/ZeroGravity-82/goph-profile/migrations"
)

func openTestDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URI")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URI is not set")
	}

	db, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	require.NoError(t, runTestMigrations(ctx, db))
	truncateTestTables(t, ctx, db)
	t.Cleanup(func() {
		truncateTestTables(t, ctx, db)
	})

	return db
}

func runTestMigrations(ctx context.Context, db *pgxpool.Pool) error {
	migrationDB := stdlib.OpenDBFromPool(db)
	defer migrationDB.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, migrationDB, migrationfiles.FS)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

func truncateTestTables(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(ctx, "TRUNCATE TABLE avatar, app_user CASCADE")
	require.NoError(t, err)
}

func newTestUser(t *testing.T, rawEmail string) model.User {
	t.Helper()

	id, err := uuid.NewV7()
	require.NoError(t, err)
	email, err := model.NewEmail(rawEmail)
	require.NoError(t, err)
	user, err := model.NewUser(id, email, fixedTestTime())
	require.NoError(t, err)
	return user
}

func newTestProcessingAvatar(t *testing.T, userID uuid.UUID, fileName string) model.Avatar {
	t.Helper()

	id, err := uuid.NewV7()
	require.NoError(t, err)
	avatar, err := model.NewProcessingAvatar(
		id,
		userID,
		fileName,
		model.MIMEPNG,
		1024,
		"users/"+userID.String()+"/avatars/"+id.String()+"/original",
		fixedTestTime(),
	)
	require.NoError(t, err)
	return avatar
}

func newTestReadyAvatar(t *testing.T, userID uuid.UUID, fileName string) model.Avatar {
	t.Helper()

	avatar := newTestProcessingAvatar(t, userID, fileName)
	err := avatar.MarkReady(
		100,
		100,
		avatar.ObjectKeyOriginal+"/thumb-100",
		avatar.ObjectKeyOriginal+"/thumb-300",
		fixedTestTime().Add(time.Minute),
	)
	require.NoError(t, err)
	return avatar
}

func fixedTestTime() time.Time {
	return time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
}
