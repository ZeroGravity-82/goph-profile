package rabbitmq

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
)

// TestConsumer_Run_WaitsForAckAfterContextCancel проверяет, что после отмены контекста Run не завершается
// до подтверждения активного сообщения.
func TestConsumer_Run_WaitsForAckAfterContextCancel(t *testing.T) {
	// Arrange
	processingChannel := newShutdownDeliverySubscriberFake()
	deletionChannel := newShutdownDeliverySubscriberFake()
	handler := newShutdownMessageHandlerFake()
	ackStarted := make(chan struct{})
	ackRelease := make(chan struct{})
	acknowledger := &shutdownAcknowledgerFake{ackStarted: ackStarted, ackRelease: ackRelease}
	processingChannel.deliveries <- shutdownProcessingDelivery(acknowledger)
	consumer := newShutdownConsumer(processingChannel, deletionChannel, handler, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	stopHandlerRelease := context.AfterFunc(ctx, func() {
		close(handler.releaseProcessing)
	})
	defer stopHandlerRelease()
	resultCh := make(chan error, 1)

	go func() {
		resultCh <- consumer.Run(ctx)
	}()
	waitForShutdownSignal(t, handler.processingStarted)

	// Act
	cancel()

	// Assert
	waitForShutdownSignal(t, ackStarted)
	assertConsumerStillRunning(t, resultCh)

	close(ackRelease)
	require.NoError(t, waitForShutdownResult(t, resultCh))
}

// TestConsumer_Run_WaitsForNackAfterContextCancelTimeout проверяет, что после отмены контекста и тайм-аута
// Run дожидается Nack отмененного сообщения.
func TestConsumer_Run_WaitsForNackAfterContextCancelTimeout(t *testing.T) {
	// Arrange
	processingChannel := newShutdownDeliverySubscriberFake()
	deletionChannel := newShutdownDeliverySubscriberFake()
	handler := newShutdownMessageHandlerFake()
	nackStarted := make(chan struct{})
	nackRelease := make(chan struct{})
	acknowledger := &shutdownAcknowledgerFake{nackStarted: nackStarted, nackRelease: nackRelease}
	processingChannel.deliveries <- shutdownProcessingDelivery(acknowledger)
	consumer := newShutdownConsumer(processingChannel, deletionChannel, handler, 20*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)

	go func() {
		resultCh <- consumer.Run(ctx)
	}()
	waitForShutdownSignal(t, handler.processingStarted)

	// Act
	cancel()

	// Assert
	waitForShutdownSignal(t, nackStarted)
	assertConsumerStillRunning(t, resultCh)

	close(nackRelease)
	err := waitForShutdownResult(t, resultCh)
	require.Error(t, err)
	assert.ErrorIs(t, err, errConsumerShutdownTimeout)
}

func newShutdownConsumer(
	processingChannel deliverySubscriber,
	deletionChannel deliverySubscriber,
	handler AvatarMessageHandler,
	shutdownTimeout time.Duration,
) *Consumer {
	return &Consumer{
		processingChannel: processingChannel,
		deletionChannel:   deletionChannel,
		cfg: Config{
			AvatarProcessingQueue: "avatar-processing",
			AvatarDeletionQueue:   "avatar-deletion",
		},
		handler:         handler,
		logger:          logging.NopLogger(),
		shutdownTimeout: shutdownTimeout,
	}
}

func shutdownProcessingDelivery(acknowledger amqp.Acknowledger) amqp.Delivery {
	return amqp.Delivery{
		Acknowledger: acknowledger,
		DeliveryTag:  1,
		Body: []byte(`{
			"avatar_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003",
			"user_id":"018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001",
			"object_key_original":"users/user-id/avatars/avatar-id/original"
		}`),
	}
}

func waitForShutdownSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("shutdown signal was not received")
	}
}

func assertConsumerStillRunning(t *testing.T, resultCh <-chan error) {
	t.Helper()

	select {
	case err := <-resultCh:
		t.Fatalf("consumer stopped before message acknowledgement completed: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
}

func waitForShutdownResult(t *testing.T, resultCh <-chan error) error {
	t.Helper()

	select {
	case err := <-resultCh:
		return err
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop")
		return nil
	}
}
