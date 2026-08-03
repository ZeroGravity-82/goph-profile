package rabbitmq

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

// TestConsumer_Run_MarksNotReadyOnContextCancel проверяет, что после отмены контекста консьюмер перестает быть готовым.
func TestConsumer_Run_MarksNotReadyOnContextCancel(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	processingChannel := newShutdownConsumerChannelFake()
	deletionChannel := newShutdownConsumerChannelFake()
	consumer := &Consumer{
		conn:              &amqp.Connection{},
		processingChannel: processingChannel,
		deletionChannel:   deletionChannel,
		cfg:               validRabbitMQConfig(),
		handler:           avatarMessageHandlerStub{},
		logger:            logging.NopLogger(),
		shutdownTimeout:   time.Second,
	}
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- consumer.Run(ctx)
	}()
	require.Eventually(t, consumer.ready.Load, time.Second, time.Millisecond)
	require.NoError(t, consumer.Ping(ctx))

	// Act
	cancel()

	// Assert
	require.NoError(t, <-resultCh)
	require.ErrorIs(t, consumer.Ping(context.Background()), errConsumerNotReady)
}
