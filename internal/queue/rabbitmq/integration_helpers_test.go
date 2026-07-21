//go:build integration

package rabbitmq

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

// newTestConfig создает конфиг с отдельными обменником и очередями для одного интеграционного теста.
func newTestConfig(t *testing.T) Config {
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

// newTestPublisher создает паблишер и регистрирует очистку обменника и очередей.
func newTestPublisher(t *testing.T, ctx context.Context, cfg Config) *Publisher {
	t.Helper()

	publisher, err := NewPublisher(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, publisher.Close())
		cleanupTopology(t, cfg)
	})
	return publisher
}

// cleanupTopology удаляет обменник и очереди, созданные для интеграционного теста.
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

// testConsumer хранит зависимости для проверки запущенного консьюмера.
type testConsumer struct {
	ctx       context.Context
	publisher *Publisher
	handler   *avatarMessageHandlerFake
}

// newTestConsumer запускает консьюмер с fake-обработчиком и паблишером для отправки сообщений в его очереди.
func newTestConsumer(t *testing.T) testConsumer {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := newTestConfig(t)
	handler := newAvatarMessageHandlerFake()
	consumer, err := NewConsumer(ctx, cfg, handler, nil)
	require.NoError(t, err)
	publisher, err := NewPublisher(ctx, cfg)
	require.NoError(t, err)
	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-errCh)
		require.NoError(t, publisher.Close())
		require.NoError(t, consumer.Close())
		cleanupTopology(t, cfg)
	})

	return testConsumer{
		ctx:       ctx,
		publisher: publisher,
		handler:   handler,
	}
}
