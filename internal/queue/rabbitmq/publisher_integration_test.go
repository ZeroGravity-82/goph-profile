//go:build integration

package rabbitmq

import (
	"context"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestPublisher_PublishesAvatarProcessingMessage проверяет доставку задачи обработки аватарки в очередь.
func TestPublisher_PublishesAvatarProcessingMessage(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := newTestConfig(t)
	publisher := newTestPublisher(t, ctx, cfg)
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
	err = publisher.PublishAvatarProcessing(ctx, message)

	// Assert
	require.NoError(t, err)
	delivery := getMessage(t, cfg.URL, cfg.AvatarProcessingQueue)
	assert.Equal(t, "application/json", delivery.ContentType)
	assert.Equal(t, uint8(amqp.Persistent), delivery.DeliveryMode)
	assert.Equal(t, "avatar-processing:"+avatarID.String(), delivery.MessageId)
	assert.JSONEq(t, `{
		"avatar_id": "`+avatarID.String()+`",
		"user_id": "`+userID.String()+`",
		"object_key_original": "users/avatar-original"
	}`, string(delivery.Body))
}

// TestPublisher_PublishesAvatarDeletionMessage проверяет доставку задачи удаления файлов аватарки в очередь.
func TestPublisher_PublishesAvatarDeletionMessage(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := newTestConfig(t)
	publisher := newTestPublisher(t, ctx, cfg)
	avatarID, err := uuid.NewV7()
	require.NoError(t, err)
	message := usecase.AvatarDeletionMessage{
		AvatarID:   avatarID,
		ObjectKeys: []string{"users/avatar-original", "users/avatar-thumb-100", "users/avatar-thumb-300"},
	}

	// Act
	err = publisher.PublishAvatarDeletion(ctx, message)

	// Assert
	require.NoError(t, err)
	delivery := getMessage(t, cfg.URL, cfg.AvatarDeletionQueue)
	assert.Equal(t, "application/json", delivery.ContentType)
	assert.Equal(t, uint8(amqp.Persistent), delivery.DeliveryMode)
	assert.Equal(t, "avatar-deletion:"+avatarID.String(), delivery.MessageId)
	assert.JSONEq(t, `{
		"avatar_id": "`+avatarID.String()+`",
		"object_keys": ["users/avatar-original", "users/avatar-thumb-100", "users/avatar-thumb-300"]
	}`, string(delivery.Body))
}

// getMessage читает одно сообщение из очереди RabbitMQ для проверки публикации.
func getMessage(t *testing.T, url string, queueName string) amqp.Delivery {
	t.Helper()

	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	channel, err := conn.Channel()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, channel.Close())
	}()

	delivery, ok, err := channel.Get(queueName, true)
	require.NoError(t, err)
	require.True(t, ok)
	return delivery
}
