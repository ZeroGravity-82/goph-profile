package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProcessingAvatar проверяет создание аватарки в статусе обработки.
func TestNewProcessingAvatar(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	// Act
	avatar, err := NewProcessingAvatar(
		"avatar-id",
		"user-id",
		"avatar.jpg",
		MIMEJPEG,
		1024,
		"users/user-id/avatars/avatar-id/original",
		now,
	)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, AvatarStatusProcessing, avatar.Status)
	assert.Equal(t, now, avatar.CreatedAt)
	assert.Equal(t, now, avatar.UpdatedAt)
}

// TestNewProcessingAvatar_RejectsTooLargeFile проверяет лимит файла.
func TestNewProcessingAvatar_RejectsTooLargeFile(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	// Act
	_, err := NewProcessingAvatar(
		"avatar-id",
		"user-id",
		"avatar.jpg",
		MIMEJPEG,
		MaxAvatarFileSizeBytes+1,
		"users/user-id/avatars/avatar-id/original",
		now,
	)

	// Assert
	require.ErrorIs(t, err, ErrFileTooLarge)
}

// TestMarkReady проверяет перевод обработанной аватарки в статус ready.
func TestMarkReady(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	readyAt := now.Add(time.Second)
	avatar := mustProcessingAvatar(t, now)

	// Act
	err := avatar.MarkReady(
		4096,
		4096,
		"thumbs/avatar-id/100.png",
		"thumbs/avatar-id/300.png",
		readyAt,
	)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, AvatarStatusReady, avatar.Status)
	assert.NotNil(t, avatar.Width)
	assert.Equal(t, 4096, *avatar.Width)
	assert.NotNil(t, avatar.Height)
	assert.Equal(t, 4096, *avatar.Height)
	assert.NotNil(t, avatar.ObjectKeyThumb100)
	assert.NotEmpty(t, *avatar.ObjectKeyThumb100)
	assert.Equal(t, readyAt, avatar.UpdatedAt)
}

// TestMarkReady_RejectsTooLargeDimensions проверяет лимит пикселей.
func TestMarkReady_RejectsTooLargeDimensions(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	avatar := mustProcessingAvatar(t, now)

	// Act
	err := avatar.MarkReady(
		MaxImageWidth+1,
		100,
		"thumbs/avatar-id/100.png",
		"thumbs/avatar-id/300.png",
		now,
	)

	// Assert
	require.ErrorIs(t, err, ErrImageTooLarge)
}

// TestCanBeCurrent проверяет правила выбора текущей аватарки.
func TestCanBeCurrent(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	ready := mustReadyAvatar(t, now)
	processing := mustProcessingAvatar(t, now)

	// Act
	readyErr := ready.CanBeCurrent()
	processingErr := processing.CanBeCurrent()
	require.NoError(t, ready.MarkDeleting(now))
	deletedErr := ready.CanBeCurrent()

	// Assert
	require.NoError(t, readyErr)
	require.ErrorIs(t, processingErr, ErrAvatarNotReady)
	require.ErrorIs(t, deletedErr, ErrAvatarDeleted)
}

// TestMarkDeleting_IsIdempotent проверяет идемпотентность мягкого удаления.
func TestMarkDeleting_IsIdempotent(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	avatar := mustProcessingAvatar(t, now)

	// Act
	firstErr := avatar.MarkDeleting(now)
	secondErr := avatar.MarkDeleting(now.Add(time.Second))

	// Assert
	require.NoError(t, firstErr)
	require.NoError(t, secondErr)
	assert.Equal(t, AvatarStatusDeleting, avatar.Status)
	require.NotNil(t, avatar.DeletedAt)
	assert.Equal(t, now, *avatar.DeletedAt)
}

func mustProcessingAvatar(t *testing.T, now time.Time) Avatar {
	t.Helper()

	avatar, err := NewProcessingAvatar(
		"avatar-id",
		"user-id",
		"avatar.png",
		MIMEPNG,
		1024,
		"users/user-id/avatars/avatar-id/original",
		now,
	)
	require.NoError(t, err)

	return avatar
}
