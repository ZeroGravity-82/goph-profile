package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// AvatarMessageHandler описывает обработчик задач аватарок из RabbitMQ.
type AvatarMessageHandler interface {
	HandleAvatarProcessing(ctx context.Context, message usecase.AvatarProcessingMessage) error
	HandleAvatarDeletion(ctx context.Context, message usecase.AvatarDeletionMessage) error
}

// Consumer читает задачи аватарок из RabbitMQ.
type Consumer struct {
	conn              *amqp.Connection
	processingChannel *amqp.Channel
	deletionChannel   *amqp.Channel
	cfg               Config
	handler           AvatarMessageHandler
	logger            *slog.Logger
}

// NewConsumer создает подключение к RabbitMQ, обменник, очереди и каналы чтения.
func NewConsumer(
	ctx context.Context,
	cfg Config,
	handler AvatarMessageHandler,
	logger *slog.Logger,
) (*Consumer, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("avatar message handler is not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to create rabbitmq consumer: %w", ctx.Err())
	default:
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	consumer, err := newConsumerWithConnection(conn, cfg, handler, logger)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return consumer, nil
}

// newConsumerWithConnection создает обменник, очереди и отдельные каналы чтения для каждой очереди.
func newConsumerWithConnection(
	conn *amqp.Connection,
	cfg Config,
	handler AvatarMessageHandler,
	logger *slog.Logger,
) (*Consumer, error) {
	setupChannel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open rabbitmq setup channel: %w", err)
	}
	if err = (&Publisher{channel: setupChannel, cfg: cfg}).declareTopology(); err != nil {
		_ = setupChannel.Close()
		return nil, err
	}
	if err = setupChannel.Close(); err != nil {
		return nil, fmt.Errorf("failed to close rabbitmq setup channel: %w", err)
	}

	processingChannel, err := consumerChannel(conn)
	if err != nil {
		return nil, err
	}
	deletionChannel, err := consumerChannel(conn)
	if err != nil {
		_ = processingChannel.Close()
		return nil, err
	}

	return &Consumer{
		conn:              conn,
		processingChannel: processingChannel,
		deletionChannel:   deletionChannel,
		cfg:               cfg,
		handler:           handler,
		logger:            logger,
	}, nil
}

// consumerChannel открывает канал RabbitMQ и ограничивает консьюмер одним неподтвержденным сообщением.
func consumerChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open rabbitmq consumer channel: %w", err)
	}
	if err = channel.Qos(1, 0, false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("failed to configure rabbitmq consumer prefetch: %w", err)
	}
	return channel, nil
}

// Run читает сообщения задач аватарок до отмены контекста или ошибки RabbitMQ.
func (c *Consumer) Run(ctx context.Context) error {
	processingDeliveryCh, err := c.processingChannel.Consume(
		c.cfg.AvatarProcessingQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to consume avatar processing queue: %w", err)
	}

	deletionDeliveryCh, err := c.deletionChannel.Consume(
		c.cfg.AvatarDeletionQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to consume avatar deletion queue: %w", err)
	}

	c.logger.InfoContext(ctx, "starting rabbitmq consumer",
		slog.String("avatar_processing_queue", c.cfg.AvatarProcessingQueue),
		slog.String("avatar_deletion_queue", c.cfg.AvatarDeletionQueue),
	)

	errCh := make(chan error, 2)
	go c.consumeProcessing(ctx, processingDeliveryCh, errCh)
	go c.consumeDeletion(ctx, deletionDeliveryCh, errCh)

	select {
	case <-ctx.Done():
		return nil
	case err = <-errCh:
		return err
	}
}

// consumeProcessing читает очередь задач обработки аватарок до отмены контекста или ошибки подтверждения сообщения.
func (c *Consumer) consumeProcessing(
	ctx context.Context,
	deliveryCh <-chan amqp.Delivery,
	errCh chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveryCh:
			if !ok {
				errCh <- errors.New("rabbitmq avatar processing delivery channel closed")
				return
			}
			if err := c.handleProcessingDelivery(ctx, delivery); err != nil {
				errCh <- err
				return
			}
		}
	}
}

// handleProcessingDelivery парсит задачу обработки, передает ее хендлеру и подтверждает результат в RabbitMQ.
//
// Невалидное сообщение отклоняется через Reject без повторной доставки. Ошибка обработчика считается временной
// ошибкой выполнения и подтверждается через Nack с возвратом сообщения в очередь.
func (c *Consumer) handleProcessingDelivery(ctx context.Context, delivery amqp.Delivery) error {
	message, err := decodeAvatarProcessingMessage(delivery.Body)
	if err != nil {
		c.logger.WarnContext(ctx, "invalid avatar processing message", slog.Any("err", err))
		return delivery.Reject(false)
	}
	if err = c.handler.HandleAvatarProcessing(ctx, message); err != nil {
		c.logger.ErrorContext(ctx, "avatar processing message failed",
			slog.String("avatar_id", message.AvatarID.String()),
			slog.Any("err", err),
		)
		return delivery.Nack(false, true)
	}
	return delivery.Ack(false)
}

// decodeAvatarProcessingMessage парсит JSON-сообщение RabbitMQ в задачу обработки аватарки.
func decodeAvatarProcessingMessage(body []byte) (usecase.AvatarProcessingMessage, error) {
	var payload avatarProcessingMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return usecase.AvatarProcessingMessage{}, fmt.Errorf("failed to decode avatar processing message: %w", err)
	}

	avatarID, err := uuid.Parse(payload.AvatarID)
	if err != nil {
		return usecase.AvatarProcessingMessage{}, fmt.Errorf("failed to parse avatar_id: %w", err)
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return usecase.AvatarProcessingMessage{}, fmt.Errorf("failed to parse user_id: %w", err)
	}
	if payload.ObjectKeyOriginal == "" {
		return usecase.AvatarProcessingMessage{}, errors.New("object_key_original is not provided")
	}

	return usecase.AvatarProcessingMessage{
		AvatarID:          avatarID,
		UserID:            userID,
		ObjectKeyOriginal: payload.ObjectKeyOriginal,
	}, nil
}

// consumeDeletion читает очередь задач удаления файлов аватарок до отмены контекста или ошибки подтверждения сообщения.
func (c *Consumer) consumeDeletion(ctx context.Context, deliveryCh <-chan amqp.Delivery, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveryCh:
			if !ok {
				errCh <- errors.New("rabbitmq avatar deletion delivery channel closed")
				return
			}
			if err := c.handleDeletionDelivery(ctx, delivery); err != nil {
				errCh <- err
				return
			}
		}
	}
}

// handleDeletionDelivery парсит задачу удаления, передает ее хендлеру и подтверждает результат в RabbitMQ.
//
// Невалидное сообщение отклоняется через Reject без повторной доставки. Ошибка обработчика считается временной
// ошибкой выполнения и подтверждается через Nack с возвратом сообщения в очередь.
func (c *Consumer) handleDeletionDelivery(ctx context.Context, delivery amqp.Delivery) error {
	message, err := decodeAvatarDeletionMessage(delivery.Body)
	if err != nil {
		c.logger.WarnContext(ctx, "invalid avatar deletion message", slog.Any("err", err))
		return delivery.Reject(false)
	}
	if err = c.handler.HandleAvatarDeletion(ctx, message); err != nil {
		c.logger.ErrorContext(ctx, "avatar deletion message failed",
			slog.String("avatar_id", message.AvatarID.String()),
			slog.Any("err", err),
		)
		return delivery.Nack(false, true)
	}
	return delivery.Ack(false)
}

// decodeAvatarDeletionMessage парсит JSON-сообщение RabbitMQ в задачу удаления файлов аватарки.
func decodeAvatarDeletionMessage(body []byte) (usecase.AvatarDeletionMessage, error) {
	var payload avatarDeletionMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return usecase.AvatarDeletionMessage{}, fmt.Errorf("failed to decode avatar deletion message: %w", err)
	}

	avatarID, err := uuid.Parse(payload.AvatarID)
	if err != nil {
		return usecase.AvatarDeletionMessage{}, fmt.Errorf("failed to parse avatar_id: %w", err)
	}
	if len(payload.ObjectKeys) == 0 {
		return usecase.AvatarDeletionMessage{}, errors.New("object_keys are not provided")
	}
	for _, objectKey := range payload.ObjectKeys {
		if objectKey == "" {
			return usecase.AvatarDeletionMessage{}, errors.New("object key is not provided")
		}
	}

	return usecase.AvatarDeletionMessage{
		AvatarID:   avatarID,
		ObjectKeys: payload.ObjectKeys,
	}, nil
}

// Close закрывает каналы чтения и соединение RabbitMQ.
func (c *Consumer) Close() error {
	var err error
	if c.processingChannel != nil {
		err = errors.Join(err, c.processingChannel.Close())
	}
	if c.deletionChannel != nil {
		err = errors.Join(err, c.deletionChannel.Close())
	}
	if c.conn != nil {
		err = errors.Join(err, c.conn.Close())
	}
	return err
}
