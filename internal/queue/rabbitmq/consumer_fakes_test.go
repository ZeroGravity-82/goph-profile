package rabbitmq

import (
	"context"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

type shutdownConsumerChannelFake struct {
	deliveries chan amqp.Delivery
	cancelOnce sync.Once
}

func newShutdownConsumerChannelFake() *shutdownConsumerChannelFake {
	return &shutdownConsumerChannelFake{deliveries: make(chan amqp.Delivery, 1)}
}

func (c *shutdownConsumerChannelFake) Consume(
	_ string,
	_ string,
	_ bool,
	_ bool,
	_ bool,
	_ bool,
	_ amqp.Table,
) (<-chan amqp.Delivery, error) {
	return c.deliveries, nil
}

func (c *shutdownConsumerChannelFake) Cancel(_ string, _ bool) error {
	c.cancelOnce.Do(func() {
		close(c.deliveries)
	})
	return nil
}

type shutdownMessageHandlerFake struct {
	processingStarted chan struct{}
	releaseProcessing chan struct{}
}

func newShutdownMessageHandlerFake() *shutdownMessageHandlerFake {
	return &shutdownMessageHandlerFake{
		processingStarted: make(chan struct{}),
		releaseProcessing: make(chan struct{}),
	}
}

func (h *shutdownMessageHandlerFake) HandleAvatarProcessing(
	ctx context.Context,
	_ usecase.AvatarProcessingMessage,
) error {
	close(h.processingStarted)
	select {
	case <-h.releaseProcessing:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *shutdownMessageHandlerFake) HandleAvatarDeletion(
	_ context.Context,
	_ usecase.AvatarDeletionMessage,
) error {
	return nil
}

type shutdownAcknowledgerFake struct {
	ackStarted  chan<- struct{}
	ackRelease  <-chan struct{}
	nackStarted chan<- struct{}
	nackRelease <-chan struct{}
}

func (a *shutdownAcknowledgerFake) Ack(_ uint64, _ bool) error {
	a.ackStarted <- struct{}{}
	<-a.ackRelease
	return nil
}

func (a *shutdownAcknowledgerFake) Nack(_ uint64, _ bool, _ bool) error {
	a.nackStarted <- struct{}{}
	<-a.nackRelease
	return nil
}

func (a *shutdownAcknowledgerFake) Reject(_ uint64, _ bool) error {
	return nil
}
