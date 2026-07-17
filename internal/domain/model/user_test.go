package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewUser проверяет создание пользователя с нормализованным email.
func TestNewUser(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	// Act
	user, err := NewUser(testUserID, "user@example.com", now)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testUserID, user.ID)
	assert.Equal(t, Email("user@example.com"), user.Email)
	assert.Nil(t, user.CurrentAvatarID)
}

// TestNewUserID проверяет генерацию UUIDv7 для пользователя.
func TestNewUserID(t *testing.T) {
	// Act
	id, err := NewUserID()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testUserID.Version(), id.Version())
}

// TestSelectCurrentAvatar проверяет выбор готовой аватарки как текущей.
func TestSelectCurrentAvatar(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	updatedAt := now.Add(time.Second)
	user, err := NewUser(testUserID, "user@example.com", now)
	require.NoError(t, err)
	avatar := mustReadyAvatar(t, now)

	// Act
	err = user.SelectCurrentAvatar(avatar, updatedAt)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user.CurrentAvatarID)
	assert.Equal(t, avatar.ID, *user.CurrentAvatarID)
	assert.Equal(t, updatedAt, user.UpdatedAt)

	// Act
	err = user.SelectCurrentAvatar(avatar, updatedAt.Add(time.Second))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, updatedAt, user.UpdatedAt)
}

// TestSelectCurrentAvatar_RejectsForeignAvatar проверяет запрет выбора чужой аватарки.
func TestSelectCurrentAvatar_RejectsForeignAvatar(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	user, err := NewUser(testUserID, "user@example.com", now)
	require.NoError(t, err)
	avatar := mustReadyAvatar(t, now)
	avatar.UserID = testOtherUserID

	// Act
	err = user.SelectCurrentAvatar(avatar, now)

	// Assert
	require.ErrorIs(t, err, ErrAvatarForbidden)
}

// TestClearCurrentAvatar проверяет сброс текущей аватарки пользователя.
func TestClearCurrentAvatar(t *testing.T) {
	// Arrange
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	updatedAt := now.Add(time.Second)
	user, err := NewUser(testUserID, "user@example.com", now)
	require.NoError(t, err)
	avatar := mustReadyAvatar(t, now)
	require.NoError(t, user.SelectCurrentAvatar(avatar, now))

	// Act
	cleared := user.ClearCurrentAvatar(avatar.ID, updatedAt)

	// Assert
	assert.True(t, cleared)
	assert.Nil(t, user.CurrentAvatarID)
	assert.Equal(t, updatedAt, user.UpdatedAt)

	// Act
	cleared = user.ClearCurrentAvatar(avatar.ID, updatedAt.Add(time.Second))

	// Assert
	assert.False(t, cleared)
}

func mustReadyAvatar(t *testing.T, now time.Time) Avatar {
	t.Helper()

	avatar := mustProcessingAvatar(t, now)
	err := avatar.MarkReady(
		100,
		100,
		"thumbs/avatar-id/100.png",
		"thumbs/avatar-id/300.png",
		now,
	)
	require.NoError(t, err)

	return avatar
}
