//go:build integration

package rabbitmq

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestConsumer_HandlesAvatarProcessingMessage проверяет передачу задачи обработки аватарки обработчику.
func TestConsumer_HandlesAvatarProcessingMessage(t *testing.T) {
	// Arrange
	consumer := newTestConsumer(t)
	avatarID, err := uuid.NewV7()
	require.NoError(t, err)
	userID, err := uuid.NewV7()
	require.NoError(t, err)
	message := usecase.AvatarProcessingMessage{
		AvatarID:          avatarID,
		UserID:            userID,
		ObjectKeyOriginal: "users/avatar-original",
	}

	// Act
	err = consumer.publisher.PublishAvatarProcessing(consumer.ctx, message)

	// Assert
	require.NoError(t, err)
	select {
	case receivedMessage := <-consumer.handler.processing:
		assert.Equal(t, message, receivedMessage)
	case <-time.After(5 * time.Second):
		t.Fatal("avatar processing message was not received")
	}
}

// TestConsumer_HandlesAvatarDeletionMessage проверяет передачу задачи удаления файлов аватарки обработчику.
func TestConsumer_HandlesAvatarDeletionMessage(t *testing.T) {
	// Arrange
	consumer := newTestConsumer(t)
	avatarID, err := uuid.NewV7()
	require.NoError(t, err)
	message := usecase.AvatarDeletionMessage{
		AvatarID:   avatarID,
		ObjectKeys: []string{"users/avatar-original", "users/avatar-thumb-100", "users/avatar-thumb-300"},
	}

	// Act
	err = consumer.publisher.PublishAvatarDeletion(consumer.ctx, message)

	// Assert
	require.NoError(t, err)
	select {
	case receivedMessage := <-consumer.handler.deletion:
		assert.Equal(t, message, receivedMessage)
	case <-time.After(5 * time.Second):
		t.Fatal("avatar deletion message was not received")
	}
}
