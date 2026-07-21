package rabbitmq

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// Test_newAvatarProcessingMessage проверяет преобразование usecase-задачи обработки аватарки в JSON-сообщение RabbitMQ.
func Test_newAvatarProcessingMessage(t *testing.T) {
	// Arrange
	message := usecase.AvatarProcessingMessage{
		AvatarID:          uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"),
		UserID:            uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001"),
		ObjectKeyOriginal: "users/user-id/avatars/avatar-id/original",
	}

	// Act
	payload := newAvatarProcessingMessage(message)

	// Assert
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
		"user_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001",
		"object_key_original":"users/user-id/avatars/avatar-id/original"
	}`, string(body))
}

// Test_avatarProcessingMessageID проверяет идентификатор RabbitMQ-сообщения обработки аватарки.
func Test_avatarProcessingMessageID(t *testing.T) {
	// Arrange
	message := usecase.AvatarProcessingMessage{
		AvatarID:          uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"),
		UserID:            uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001"),
		ObjectKeyOriginal: "users/user-id/avatars/avatar-id/original",
	}

	// Act
	messageID := avatarProcessingMessageID(message)

	// Assert
	assert.Equal(t, "avatar-processing:018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003", messageID)
}

// Test_newAvatarDeletionMessage проверяет преобразование usecase-задачи удаления аватарки в JSON-сообщение RabbitMQ.
func Test_newAvatarDeletionMessage(t *testing.T) {
	// Arrange
	message := usecase.AvatarDeletionMessage{
		AvatarID: uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"),
		ObjectKeys: []string{
			"users/user-id/avatars/avatar-id/original",
			"users/user-id/avatars/avatar-id/thumb-100.png",
		},
	}

	// Act
	payload := newAvatarDeletionMessage(message)

	// Assert
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
		"object_keys":[
			"users/user-id/avatars/avatar-id/original",
			"users/user-id/avatars/avatar-id/thumb-100.png"
		]
	}`, string(body))
}

// Test_avatarDeletionMessageID проверяет идентификатор RabbitMQ-сообщения удаления аватарки.
func Test_avatarDeletionMessageID(t *testing.T) {
	// Arrange
	message := usecase.AvatarDeletionMessage{
		AvatarID: uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003"),
		ObjectKeys: []string{
			"users/user-id/avatars/avatar-id/original",
			"users/user-id/avatars/avatar-id/thumb-100.png",
		},
	}

	// Act
	messageID := avatarDeletionMessageID(message)

	// Assert
	assert.Equal(t, "avatar-deletion:018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003", messageID)
}
