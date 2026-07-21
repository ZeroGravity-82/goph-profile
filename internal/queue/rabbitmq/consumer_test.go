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
