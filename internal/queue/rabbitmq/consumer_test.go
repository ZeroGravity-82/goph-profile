package rabbitmq

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestDecodeAvatarProcessingMessage проверяет преобразование JSON-сообщения обработки аватарки в usecase-задачу.
func TestDecodeAvatarProcessingMessage(t *testing.T) {
	// Arrange
	body := []byte(`{
		"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
		"user_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001",
		"object_key_original":"users/user-id/avatars/avatar-id/original"
	}`)

	// Act
	message, err := decodeAvatarProcessingMessage(body)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, usecase.AvatarProcessingMessage{
		AvatarID:          uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"),
		UserID:            uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001"),
		ObjectKeyOriginal: "users/user-id/avatars/avatar-id/original",
	}, message)
}

// TestDecodeAvatarProcessingMessage_RequiresObjectKey проверяет обязательность ключа объекта исходного файла.
func TestDecodeAvatarProcessingMessage_RequiresObjectKey(t *testing.T) {
	// Arrange
	body := []byte(`{
		"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
		"user_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001"
	}`)

	// Act
	message, err := decodeAvatarProcessingMessage(body)

	// Assert
	require.EqualError(t, err, "object_key_original is not provided")
	assert.Zero(t, message)
}

// TestDecodeAvatarProcessingMessage_RejectsInvalidJSON проверяет ошибку парсинга JSON.
func TestDecodeAvatarProcessingMessage_RejectsInvalidJSON(t *testing.T) {
	// Arrange
	body := []byte(`{`)

	// Act
	message, err := decodeAvatarProcessingMessage(body)

	// Assert
	require.ErrorContains(t, err, "failed to decode avatar processing message")
	assert.Zero(t, message)
}

// TestDecodeAvatarProcessingMessage_RejectsInvalidAvatarID проверяет ошибку парсинга avatar_id.
func TestDecodeAvatarProcessingMessage_RejectsInvalidAvatarID(t *testing.T) {
	// Arrange
	body := []byte(`{
		"avatar_id":"not-a-uuid",
		"user_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001",
		"object_key_original":"users/user-id/avatars/avatar-id/original"
	}`)

	// Act
	message, err := decodeAvatarProcessingMessage(body)

	// Assert
	require.ErrorContains(t, err, "failed to parse avatar_id")
	assert.Zero(t, message)
}

// TestDecodeAvatarProcessingMessage_RejectsInvalidUserID проверяет ошибку парсинга user_id.
func TestDecodeAvatarProcessingMessage_RejectsInvalidUserID(t *testing.T) {
	// Arrange
	body := []byte(`{
		"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
		"user_id":"not-a-uuid",
		"object_key_original":"users/user-id/avatars/avatar-id/original"
	}`)

	// Act
	message, err := decodeAvatarProcessingMessage(body)

	// Assert
	require.ErrorContains(t, err, "failed to parse user_id")
	assert.Zero(t, message)
}

// TestDecodeAvatarDeletionMessage проверяет преобразование JSON-сообщения удаления аватарки в usecase-задачу.
func TestDecodeAvatarDeletionMessage(t *testing.T) {
	// Arrange
	body := []byte(`{
		"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
		"object_keys":["users/user-id/avatars/avatar-id/original","users/user-id/avatars/avatar-id/thumb-100.png"]
	}`)

	// Act
	message, err := decodeAvatarDeletionMessage(body)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, usecase.AvatarDeletionMessage{
		AvatarID: uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"),
		ObjectKeys: []string{
			"users/user-id/avatars/avatar-id/original",
			"users/user-id/avatars/avatar-id/thumb-100.png",
		},
	}, message)
}

// TestDecodeAvatarDeletionMessage_RequiresObjectKeys проверяет обязательность ключей объектов удаляемых файлов.
func TestDecodeAvatarDeletionMessage_RequiresObjectKeys(t *testing.T) {
	// Arrange
	body := []byte(`{"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"}`)

	// Act
	message, err := decodeAvatarDeletionMessage(body)

	// Assert
	require.EqualError(t, err, "object_keys are not provided")
	assert.Zero(t, message)
}

// TestDecodeAvatarDeletionMessage_RejectsInvalidJSON проверяет ошибку парсинга JSON.
func TestDecodeAvatarDeletionMessage_RejectsInvalidJSON(t *testing.T) {
	// Arrange
	body := []byte(`{`)

	// Act
	message, err := decodeAvatarDeletionMessage(body)

	// Assert
	require.ErrorContains(t, err, "failed to decode avatar deletion message")
	assert.Zero(t, message)
}

// TestDecodeAvatarDeletionMessage_RejectsInvalidAvatarID проверяет ошибку парсинга avatar_id.
func TestDecodeAvatarDeletionMessage_RejectsInvalidAvatarID(t *testing.T) {
	// Arrange
	body := []byte(`{"avatar_id":"not-a-uuid","object_keys":["users/user-id/avatars/avatar-id/original"]}`)

	// Act
	message, err := decodeAvatarDeletionMessage(body)

	// Assert
	require.ErrorContains(t, err, "failed to parse avatar_id")
	assert.Zero(t, message)
}

// TestDecodeAvatarDeletionMessage_RequiresNonEmptyObjectKey проверяет обязательность каждого ключа объекта.
func TestDecodeAvatarDeletionMessage_RequiresNonEmptyObjectKey(t *testing.T) {
	// Arrange
	body := []byte(`{"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003","object_keys":[""]}`)

	// Act
	message, err := decodeAvatarDeletionMessage(body)

	// Assert
	require.EqualError(t, err, "object key is not provided")
	assert.Zero(t, message)
}
