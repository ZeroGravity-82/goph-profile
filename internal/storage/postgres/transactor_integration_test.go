//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// TestTransactor_WithinTransaction_Commit проверяет фиксацию изменений после успешного выполнения callback.
func TestTransactor_WithinTransaction_Commit(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	transactor, err := NewTransactor(db)
	require.NoError(t, err)
	userRepo, err := NewUserRepository(db)
	require.NoError(t, err)
	avatarRepo, err := NewAvatarRepository(db)
	require.NoError(t, err)
	email, err := model.NewEmail("tx-commit@example.com")
	require.NoError(t, err)
	var user model.User
	var avatar model.Avatar

	// Act
	err = transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		user, err = userRepo.Create(ctx, email)
		if err != nil {
			return err
		}
		avatar = newTestReadyAvatar(t, user.ID, "tx-commit.png")
		return avatarRepo.Create(ctx, avatar)
	})

	// Assert
	require.NoError(t, err)

	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotUser.ID)

	gotAvatar, err := avatarRepo.GetByID(ctx, avatar.ID)
	require.NoError(t, err)
	assert.Equal(t, avatar.ID, gotAvatar.ID)
}

// TestTransactor_WithinTransaction_Rollback проверяет откат изменений, если callback вернул ошибку.
func TestTransactor_WithinTransaction_Rollback(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	transactor, err := NewTransactor(db)
	require.NoError(t, err)
	userRepo, err := NewUserRepository(db)
	require.NoError(t, err)
	avatarRepo, err := NewAvatarRepository(db)
	require.NoError(t, err)
	email, err := model.NewEmail("tx-rollback@example.com")
	require.NoError(t, err)
	wantErr := errors.New("fail transaction")
	var avatar model.Avatar

	// Act
	err = transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		user, err := userRepo.Create(ctx, email)
		if err != nil {
			return err
		}
		avatar = newTestReadyAvatar(t, user.ID, "tx-rollback.png")
		if err = avatarRepo.Create(ctx, avatar); err != nil {
			return err
		}
		return wantErr
	})

	// Assert
	require.ErrorIs(t, err, wantErr)

	_, err = userRepo.GetByEmail(ctx, email)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrUserNotFound))

	_, err = avatarRepo.GetByID(ctx, avatar.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrAvatarNotFound))
}
