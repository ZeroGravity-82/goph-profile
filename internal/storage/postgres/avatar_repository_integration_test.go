//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestAvatarRepository_Create проверяет создание аватарки.
func TestAvatarRepository_Create(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo, avatarRepo := newTestAvatarRepositories(t, ctx)
	user, err := userRepo.Create(ctx, mustTestEmail(t, "avatar-create@example.com"))
	require.NoError(t, err)
	avatar := newTestReadyAvatar(t, user.ID, "created.png")

	// Act
	err = avatarRepo.Create(ctx, avatar)

	// Assert
	require.NoError(t, err)
}

// TestAvatarRepository_GetByID проверяет получение аватарки по ID.
func TestAvatarRepository_GetByID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo, avatarRepo := newTestAvatarRepositories(t, ctx)
	user, err := userRepo.Create(ctx, mustTestEmail(t, "avatar-get@example.com"))
	require.NoError(t, err)
	avatar := newTestReadyAvatar(t, user.ID, "get.png")
	require.NoError(t, avatarRepo.Create(ctx, avatar))

	// Act
	got, err := avatarRepo.GetByID(ctx, avatar.ID)

	// Assert
	require.NoError(t, err)
	assertAvatarEqual(t, avatar, got)
}

func newTestAvatarRepositories(t *testing.T, ctx context.Context) (*UserRepository, *AvatarRepository) {
	t.Helper()

	db := openTestDB(t, ctx)
	userRepo, err := NewUserRepository(db)
	require.NoError(t, err)
	avatarRepo, err := NewAvatarRepository(db)
	require.NoError(t, err)
	return userRepo, avatarRepo
}

func mustTestEmail(t *testing.T, raw string) model.Email {
	t.Helper()

	email, err := model.NewEmail(raw)
	require.NoError(t, err)
	return email
}

func assertAvatarEqual(t *testing.T, want model.Avatar, got model.Avatar) {
	t.Helper()

	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.FileName, got.FileName)
	assert.Equal(t, want.MIMEType, got.MIMEType)
	assert.Equal(t, want.SizeBytes, got.SizeBytes)
	assert.Equal(t, want.Width, got.Width)
	assert.Equal(t, want.Height, got.Height)
	assert.Equal(t, want.ObjectKeyOriginal, got.ObjectKeyOriginal)
	assert.Equal(t, want.ObjectKeyThumb100, got.ObjectKeyThumb100)
	assert.Equal(t, want.ObjectKeyThumb300, got.ObjectKeyThumb300)
	assert.Equal(t, want.Status, got.Status)
	assert.True(t, got.CreatedAt.Equal(want.CreatedAt))
	assert.True(t, got.UpdatedAt.Equal(want.UpdatedAt))
	if want.DeletedAt == nil {
		assert.Nil(t, got.DeletedAt)
		return
	}
	require.NotNil(t, got.DeletedAt)
	assert.True(t, got.DeletedAt.Equal(*want.DeletedAt))
}

// TestAvatarRepository_ListByUserID проверяет список неудаленных аватарок пользователя.
func TestAvatarRepository_ListByUserID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo, avatarRepo := newTestAvatarRepositories(t, ctx)
	user, err := userRepo.Create(ctx, mustTestEmail(t, "avatar-list@example.com"))
	require.NoError(t, err)
	foreignUser, err := userRepo.Create(ctx, mustTestEmail(t, "foreign-list@example.com"))
	require.NoError(t, err)
	oldAvatar := newTestReadyAvatar(t, user.ID, "old.png")
	newAvatar := newTestReadyAvatar(t, user.ID, "new.png")
	processingAvatar := newTestProcessingAvatar(t, user.ID, "processing.png")
	deletingAvatar := newTestReadyAvatar(t, user.ID, "deleting.png")
	deletedAvatar := newTestReadyAvatar(t, user.ID, "deleted.png")
	foreignAvatar := newTestReadyAvatar(t, foreignUser.ID, "foreign.png")
	oldAvatar.CreatedAt = fixedTestTime().AddDate(0, 0, -1)
	oldAvatar.UpdatedAt = oldAvatar.CreatedAt
	newAvatar.CreatedAt = fixedTestTime().AddDate(0, 0, 1)
	newAvatar.UpdatedAt = newAvatar.CreatedAt
	require.NoError(t, deletingAvatar.MarkDeleting(fixedTestTime().AddDate(0, 0, 2)))
	require.NoError(t, deletedAvatar.MarkDeleting(fixedTestTime().AddDate(0, 0, 3)))
	require.NoError(t, deletedAvatar.MarkDeleted(fixedTestTime().AddDate(0, 0, 4)))
	for _, avatar := range []model.Avatar{
		oldAvatar,
		newAvatar,
		processingAvatar,
		deletingAvatar,
		deletedAvatar,
		foreignAvatar,
	} {
		require.NoError(t, avatarRepo.Create(ctx, avatar))
	}

	// Act
	avatars, err := avatarRepo.ListByUserID(ctx, user.ID)

	// Assert
	require.NoError(t, err)
	require.Len(t, avatars, 3)
	assert.Equal(t, newAvatar.ID, avatars[0].ID)
	assert.Equal(t, processingAvatar.ID, avatars[1].ID)
	assert.Equal(t, oldAvatar.ID, avatars[2].ID)
}

// TestAvatarRepository_Update проверяет сохранение изменяемых полей аватарки.
func TestAvatarRepository_Update(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo, avatarRepo := newTestAvatarRepositories(t, ctx)
	user, err := userRepo.Create(ctx, mustTestEmail(t, "avatar-update@example.com"))
	require.NoError(t, err)
	avatar := newTestProcessingAvatar(t, user.ID, "before.png")
	require.NoError(t, avatarRepo.Create(ctx, avatar))
	require.NoError(t, avatar.MarkReady(
		300,
		300,
		"updated/thumb-100",
		"updated/thumb-300",
		fixedTestTime().AddDate(0, 0, 1),
	))
	avatar.FileName = "after.png"
	avatar.SizeBytes = 2048

	// Act
	err = avatarRepo.Update(ctx, avatar)

	// Assert
	require.NoError(t, err)
	got, err := avatarRepo.GetByID(ctx, avatar.ID)
	require.NoError(t, err)
	assertAvatarEqual(t, avatar, got)
}

// TestAvatarRepository_Delete проверяет удаление записи аватарки из таблицы.
func TestAvatarRepository_Delete(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userRepo, avatarRepo := newTestAvatarRepositories(t, ctx)
	user, err := userRepo.Create(ctx, mustTestEmail(t, "avatar-delete@example.com"))
	require.NoError(t, err)
	avatar := newTestReadyAvatar(t, user.ID, "delete.png")
	require.NoError(t, avatarRepo.Create(ctx, avatar))

	// Act
	err = avatarRepo.Delete(ctx, avatar.ID)

	// Assert
	require.NoError(t, err)
	_, err = avatarRepo.GetByID(ctx, avatar.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, usecase.ErrAvatarNotFound))
}

// TestAvatarRepository_GetByID_NotFound проверяет ошибку при поиске несуществующей аватарки.
func TestAvatarRepository_GetByID_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, avatarRepo := newTestAvatarRepositories(t, ctx)
	avatar := newTestReadyAvatar(t, newTestUser(t, "missing-avatar-user@example.com").ID, "missing.png")

	// Act
	_, err := avatarRepo.GetByID(ctx, avatar.ID)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, usecase.ErrAvatarNotFound))
}

// TestAvatarRepository_Update_NotFound проверяет ошибку при обновлении несуществующей аватарки.
func TestAvatarRepository_Update_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, avatarRepo := newTestAvatarRepositories(t, ctx)
	avatar := newTestReadyAvatar(t, newTestUser(t, "missing-update-avatar-user@example.com").ID, "missing.png")

	// Act
	err := avatarRepo.Update(ctx, avatar)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, usecase.ErrAvatarNotFound))
}

// TestAvatarRepository_Delete_NotFound проверяет ошибку при удалении несуществующей аватарки.
func TestAvatarRepository_Delete_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	_, avatarRepo := newTestAvatarRepositories(t, ctx)
	avatar := newTestReadyAvatar(t, newTestUser(t, "missing-delete-avatar-user@example.com").ID, "missing.png")

	// Act
	err := avatarRepo.Delete(ctx, avatar.ID)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, usecase.ErrAvatarNotFound))
}
