//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// TestUserRepository_Create проверяет создание пользователя.
func TestUserRepository_Create(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	repo, err := NewUserRepository(db)
	require.NoError(t, err)
	email, err := model.NewEmail("alice@example.com")
	require.NoError(t, err)

	// Act
	created, err := repo.Create(ctx, email)

	// Assert
	require.NoError(t, err)
	assert.NotEqual(t, model.User{}, created)
	assert.Equal(t, email, created.Email)
	assert.Nil(t, created.CurrentAvatarID)
}

// TestUserRepository_GetByEmail проверяет получение пользователя по email.
func TestUserRepository_GetByEmail(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	repo, err := NewUserRepository(db)
	require.NoError(t, err)
	email, err := model.NewEmail("alice-get@example.com")
	require.NoError(t, err)
	created, err := repo.Create(ctx, email)
	require.NoError(t, err)

	// Act
	gotByEmail, err := repo.GetByEmail(ctx, email)

	// Assert
	require.NoError(t, err)
	assertUserEqual(t, created, gotByEmail)
}

func assertUserEqual(t *testing.T, want model.User, got model.User) {
	t.Helper()

	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.Email, got.Email)
	assert.Equal(t, want.CurrentAvatarID, got.CurrentAvatarID)
	assert.WithinDuration(t, want.CreatedAt, got.CreatedAt, time.Millisecond)
	assert.WithinDuration(t, want.UpdatedAt, got.UpdatedAt, time.Millisecond)
}

// TestUserRepository_Update проверяет сохранение изменяемых полей пользователя.
func TestUserRepository_Update(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	userRepo, err := NewUserRepository(db)
	require.NoError(t, err)
	avatarRepo, err := NewAvatarRepository(db)
	require.NoError(t, err)
	email, err := model.NewEmail("update@example.com")
	require.NoError(t, err)
	user, err := userRepo.Create(ctx, email)
	require.NoError(t, err)
	avatar := newTestReadyAvatar(t, user.ID, "avatar.png")
	require.NoError(t, avatarRepo.Create(ctx, avatar))
	user.CurrentAvatarID = &avatar.ID
	user.UpdatedAt = fixedTestTime().AddDate(0, 0, 1)

	// Act
	err = userRepo.Update(ctx, user)

	// Assert
	require.NoError(t, err)
	got, err := userRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assertUserEqual(t, user, got)
}

// TestUserRepository_GetByID_NotFound проверяет ошибку при поиске несуществующего пользователя по ID.
func TestUserRepository_GetByID_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	repo, err := NewUserRepository(db)
	require.NoError(t, err)
	missingUser := newTestUser(t, "missing@example.com")

	// Act
	_, err = repo.GetByID(ctx, missingUser.ID)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrUserNotFound))
}

// TestUserRepository_GetByEmail_NotFound проверяет ошибку при поиске несуществующего пользователя по email.
func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	repo, err := NewUserRepository(db)
	require.NoError(t, err)
	email, err := model.NewEmail("missing@example.com")
	require.NoError(t, err)

	// Act
	_, err = repo.GetByEmail(ctx, email)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrUserNotFound))
}

// TestUserRepository_Create_DuplicateEmail проверяет маппинг нарушения уникальности email в ошибку usecase.
func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	repo, err := NewUserRepository(db)
	require.NoError(t, err)
	email, err := model.NewEmail("duplicate@example.com")
	require.NoError(t, err)
	_, err = repo.Create(ctx, email)
	require.NoError(t, err)

	// Act
	_, err = repo.Create(ctx, email)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrEmailAlreadyTaken))
}

// TestUserRepository_Update_NotFound проверяет ошибку при обновлении несуществующего пользователя.
func TestUserRepository_Update_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	db := openTestDB(t, ctx)
	repo, err := NewUserRepository(db)
	require.NoError(t, err)
	user := newTestUser(t, "missing-update@example.com")

	// Act
	err = repo.Update(ctx, user)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrUserNotFound))
}
