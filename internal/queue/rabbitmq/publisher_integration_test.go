//go:build integration

package rabbitmq

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestPublisher_PublishAvatarProcessing_Integration проверяет публикацию задачи обработки аватарки в RabbitMQ.
func TestPublisher_PublishAvatarProcessing_Integration(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := newIntegrationConfig(t)
	publisher := newIntegrationPublisher(t, ctx, cfg)
	avatarID := uuid.New()
	userID := uuid.New()
	message := usecase.AvatarProcessingMessage{
		AvatarID:          avatarID,
		UserID:            userID,
		ObjectKeyOriginal: "users/avatar-original",
	}

	// Act
	err := publisher.PublishAvatarProcessing(ctx, message)

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

// TestPublisher_PublishAvatarDeletion_Integration проверяет публикацию задачи удаления файлов аватарки в RabbitMQ.
func TestPublisher_PublishAvatarDeletion_Integration(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := newIntegrationConfig(t)
	publisher := newIntegrationPublisher(t, ctx, cfg)
	avatarID := uuid.New()
	message := usecase.AvatarDeletionMessage{
		AvatarID:   avatarID,
		ObjectKeys: []string{"users/avatar-original", "users/avatar-thumb-100", "users/avatar-thumb-300"},
	}

	// Act
	err := publisher.PublishAvatarDeletion(ctx, message)

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

func newIntegrationConfig(t *testing.T) Config {
	t.Helper()

	url := os.Getenv("TEST_QUEUE_URL")
	if url == "" {
		t.Skip("TEST_QUEUE_URL is not set")
	}

	suffix := uuid.NewString()
	return Config{
		URL:                        url,
		Exchange:                   "goph-profile.integration.avatar." + suffix,
		AvatarProcessingQueue:      "goph-profile.integration.avatar-processing." + suffix,
		AvatarDeletionQueue:        "goph-profile.integration.avatar-deletion." + suffix,
		AvatarProcessingRoutingKey: "avatar.process",
		AvatarDeletionRoutingKey:   "avatar.delete",
	}
}

func newIntegrationPublisher(t *testing.T, ctx context.Context, cfg Config) *Publisher {
	t.Helper()

	publisher, err := NewPublisher(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, publisher.Close())
		cleanupTopology(t, cfg)
	})
	return publisher
}

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

func cleanupTopology(t *testing.T, cfg Config) {
	t.Helper()

	conn, err := amqp.Dial(cfg.URL)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	channel, err := conn.Channel()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, channel.Close())
	}()

	_, err = channel.QueueDelete(cfg.AvatarProcessingQueue, false, false, false)
	require.NoError(t, err)
	_, err = channel.QueueDelete(cfg.AvatarDeletionQueue, false, false, false)
	require.NoError(t, err)
	require.NoError(t, channel.ExchangeDelete(cfg.Exchange, false, false))
}
