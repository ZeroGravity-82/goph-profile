package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

const (
	avatarProcessingConsumerTag = "goph-profile-avatar-processing"
	avatarDeletionConsumerTag   = "goph-profile-avatar-deletion"
	consumerShutdownTimeout     = 20 * time.Second
	consumerCloseTimeout        = 5 * time.Second
)

var (
	errConsumerNotReady        = errors.New("rabbitmq consumer is not ready")
	errConsumerShutdownTimeout = errors.New("rabbitmq consumer shutdown timeout")
)

// AvatarMessageHandler описывает обработчик задач аватарок из RabbitMQ.
type AvatarMessageHandler interface {
	HandleAvatarProcessing(ctx context.Context, message usecase.AvatarProcessingMessage) error
	HandleAvatarDeletion(ctx context.Context, message usecase.AvatarDeletionMessage) error
}

// consumerChannel описывает операции RabbitMQ-канала, необходимые для получения, проверки состояния и корректной
// остановки доставок.
type consumerChannel interface {
	Consume(
		queue string,
		consumer string,
		autoAck bool,
		exclusive bool,
		noLocal bool,
		noWait bool,
		args amqp.Table,
	) (<-chan amqp.Delivery, error)
	Cancel(consumer string, noWait bool) error
	IsClosed() bool
}

// Consumer читает задачи аватарок из RabbitMQ.
type Consumer struct {
	conn              *amqp.Connection
	processingChannel consumerChannel
	deletionChannel   consumerChannel
	cfg               Config
	handler           AvatarMessageHandler
	logger            *slog.Logger
	shutdownTimeout   time.Duration
	ready             atomic.Bool
}

// NewConsumer создает подключение к RabbitMQ, объявляет топологию и открывает каналы чтения.
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

// newConsumerWithConnection объявляет топологию RabbitMQ и открывает отдельный канал чтения для каждой очереди.
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

	processingChannel, err := newConsumerChannel(conn)
	if err != nil {
		return nil, err
	}
	deletionChannel, err := newConsumerChannel(conn)
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
		logger:            logger.With("component", "rabbitmq.consumer"),
		shutdownTimeout:   consumerShutdownTimeout,
	}, nil
}

// newConsumerChannel создает новый канал RabbitMQ и ограничивает консьюмер одним неподтвержденным сообщением.
func newConsumerChannel(conn *amqp.Connection) (*amqp.Channel, error) {
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

// Run читает сообщения из обеих очередей до отмены контекста, закрытия канала доставок или ошибки RabbitMQ.
// При отмене контекста прекращает новые доставки и ожидает завершения активной обработки. Если обработка не
// завершается за shutdownTimeout, Run отменяет ее контекст и продолжает ждать завершения.
func (c *Consumer) Run(ctx context.Context) error {
	processingDeliveryCh, err := c.processingChannel.Consume(
		c.cfg.AvatarProcessingQueue,
		avatarProcessingConsumerTag,
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
		avatarDeletionConsumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		cancelErr := c.processingChannel.Cancel(avatarProcessingConsumerTag, false)
		return errors.Join(
			fmt.Errorf("failed to consume avatar deletion queue: %w", err),
			wrapConsumerCancelError("avatar processing", cancelErr),
		)
	}
	c.ready.Store(true)
	defer c.ready.Store(false)

	c.logger.InfoContext(ctx, "starting rabbitmq consumer",
		slog.String("avatar_processing_queue", c.cfg.AvatarProcessingQueue),
		slog.String("avatar_deletion_queue", c.cfg.AvatarDeletionQueue),
	)

	workCtx, cancelWork := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelWork()

	processingResultCh := make(chan error, 1)
	deletionResultCh := make(chan error, 1)
	go func() {
		processingResultCh <- c.consumeProcessing(workCtx, processingDeliveryCh)
	}()
	go func() {
		deletionResultCh <- c.consumeDeletion(workCtx, deletionDeliveryCh)
	}()

	select {
	case <-ctx.Done():
		c.ready.Store(false)
		c.logger.InfoContext(ctx, "stopping rabbitmq consumer gracefully")
		return c.stopAndWait(processingResultCh, deletionResultCh, cancelWork)
	case err = <-processingResultCh:
		cancelWork()
		return errors.Join(
			unexpectedConsumerStopError("avatar processing", err),
			c.stopAndWait(nil, deletionResultCh, cancelWork),
		)
	case err = <-deletionResultCh:
		cancelWork()
		return errors.Join(
			unexpectedConsumerStopError("avatar deletion", err),
			c.stopAndWait(processingResultCh, nil, cancelWork),
		)
	}
}

// Ping проверяет, что консьюмер запущен, а подключение и оба канала RabbitMQ открыты.
func (c *Consumer) Ping(ctx context.Context) error {
	ctx, span := startSpan(ctx, "rabbitmq.consumer.ping", "ping", c.cfg.Exchange, "", trace.SpanKindClient)
	defer span.end()

	select {
	case <-ctx.Done():
		err := fmt.Errorf("failed to ping rabbitmq consumer: %w", ctx.Err())
		span.recordError(err)
		return err
	default:
	}

	if !c.ready.Load() {
		span.recordError(errConsumerNotReady)
		return errConsumerNotReady
	}
	if c.conn == nil || c.conn.IsClosed() {
		err := errors.New("rabbitmq connection is closed")
		span.recordError(err)
		return err
	}
	if c.processingChannel == nil || c.processingChannel.IsClosed() {
		err := errors.New("rabbitmq avatar processing channel is closed")
		span.recordError(err)
		return err
	}
	if c.deletionChannel == nil || c.deletionChannel.IsClosed() {
		err := errors.New("rabbitmq avatar deletion channel is closed")
		span.recordError(err)
		return err
	}
	return nil
}

// stopAndWait останавливает консьюмеры обеих очередей и ожидает завершения оставшихся горутин.
// Если тайм-аут истекает, функция отменяет контекст активных обработчиков, но продолжает ожидание.
func (c *Consumer) stopAndWait(
	processingResultCh <-chan error,
	deletionResultCh <-chan error,
	cancelWork context.CancelFunc,
) error {
	timeout := c.shutdownTimeout
	if timeout <= 0 {
		timeout = consumerShutdownTimeout
	}
	timedOutCh := make(chan struct{})
	timer := time.AfterFunc(timeout, func() {
		cancelWork()
		close(timedOutCh)
	})

	cancelErr := c.cancelConsumers()
	processingErr := waitForConsumeLoop("avatar processing", processingResultCh)
	deletionErr := waitForConsumeLoop("avatar deletion", deletionResultCh)
	shutdownErr := errors.Join(cancelErr, processingErr, deletionErr)
	if timer.Stop() {
		return shutdownErr
	}

	<-timedOutCh
	return errors.Join(errConsumerShutdownTimeout, shutdownErr)
}

// cancelConsumers параллельно останавливает новые доставки из обеих очередей и дожидается ответа RabbitMQ.
func (c *Consumer) cancelConsumers() error {
	cancelCh := make(chan error, 2)
	go func() {
		cancelCh <- wrapConsumerCancelError(
			"avatar processing",
			c.processingChannel.Cancel(avatarProcessingConsumerTag, false),
		)
	}()
	go func() {
		cancelCh <- wrapConsumerCancelError(
			"avatar deletion",
			c.deletionChannel.Cancel(avatarDeletionConsumerTag, false),
		)
	}()

	return errors.Join(<-cancelCh, <-cancelCh)
}

// waitForConsumeLoop ожидает завершения горутины консьюмера и добавляет его имя к возникшей ошибке.
func waitForConsumeLoop(name string, resultCh <-chan error) error {
	if resultCh == nil {
		return nil
	}
	if err := <-resultCh; err != nil {
		return fmt.Errorf("rabbitmq %s consumer failed during shutdown: %w", name, err)
	}
	return nil
}

// unexpectedConsumerStopError описывает неожиданное завершение консьюмера.
func unexpectedConsumerStopError(name string, err error) error {
	if err != nil {
		return fmt.Errorf("rabbitmq %s consumer stopped: %w", name, err)
	}
	return fmt.Errorf("rabbitmq %s delivery channel closed", name)
}

// wrapConsumerCancelError добавляет к ошибке остановки имя консьюмера RabbitMQ.
func wrapConsumerCancelError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to cancel rabbitmq %s consumer: %w", name, err)
}

// consumeProcessing обрабатывает доставки задач обработки аватарок до закрытия канала доставок или ошибки
// Ack, Nack либо Reject.
func (c *Consumer) consumeProcessing(
	ctx context.Context,
	deliveryCh <-chan amqp.Delivery,
) error {
	for delivery := range deliveryCh {
		if err := c.handleProcessingDelivery(ctx, delivery); err != nil {
			return err
		}
	}
	return nil
}

// handleProcessingDelivery парсит задачу обработки, передает ее хендлеру и подтверждает результат в RabbitMQ.
//
// Невалидное сообщение отклоняется через Reject без повторной доставки. Ошибка обработчика считается временной
// ошибкой выполнения, поэтому сообщение отклоняется через Nack с возвратом в очередь.
func (c *Consumer) handleProcessingDelivery(ctx context.Context, delivery amqp.Delivery) error {
	ctx, span := startConsumeSpanWithExtractedTraceContext(ctx, delivery, "rabbitmq.consume.avatar_processing")
	defer span.end()

	message, err := decodeAvatarProcessingMessage(delivery.Body)
	if err != nil {
		span.recordError(err)
		c.logger.WarnContext(ctx, "invalid avatar processing message", slog.Any("err", err))
		if rejectErr := delivery.Reject(false); rejectErr != nil {
			span.recordError(rejectErr)
			return rejectErr
		}
		return nil
	}
	span.setAvatarID(message.AvatarID.String())
	if err = c.handler.HandleAvatarProcessing(ctx, message); err != nil {
		span.recordError(err)
		c.logger.ErrorContext(ctx, "avatar processing message failed",
			slog.String("avatar_id", message.AvatarID.String()),
			slog.Any("err", err),
		)
		if nackErr := delivery.Nack(false, true); nackErr != nil {
			span.recordError(nackErr)
			return nackErr
		}
		return nil
	}
	if ackErr := delivery.Ack(false); ackErr != nil {
		span.recordError(ackErr)
		return ackErr
	}
	return nil
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

// consumeDeletion обрабатывает доставки задач удаления файлов аватарок до закрытия канала доставок или ошибки
// Ack, Nack либо Reject.
func (c *Consumer) consumeDeletion(ctx context.Context, deliveryCh <-chan amqp.Delivery) error {
	for delivery := range deliveryCh {
		if err := c.handleDeletionDelivery(ctx, delivery); err != nil {
			return err
		}
	}
	return nil
}

// handleDeletionDelivery парсит задачу удаления, передает ее хендлеру и подтверждает результат в RabbitMQ.
//
// Невалидное сообщение отклоняется через Reject без повторной доставки. Ошибка обработчика считается временной
// ошибкой выполнения, поэтому сообщение отклоняется через Nack с возвратом в очередь.
func (c *Consumer) handleDeletionDelivery(ctx context.Context, delivery amqp.Delivery) error {
	ctx, span := startConsumeSpanWithExtractedTraceContext(ctx, delivery, "rabbitmq.consume.avatar_deletion")
	defer span.end()

	message, err := decodeAvatarDeletionMessage(delivery.Body)
	if err != nil {
		span.recordError(err)
		c.logger.WarnContext(ctx, "invalid avatar deletion message", slog.Any("err", err))
		if rejectErr := delivery.Reject(false); rejectErr != nil {
			span.recordError(rejectErr)
			return rejectErr
		}
		return nil
	}
	span.setAvatarID(message.AvatarID.String())
	if err = c.handler.HandleAvatarDeletion(ctx, message); err != nil {
		span.recordError(err)
		c.logger.ErrorContext(ctx, "avatar deletion message failed",
			slog.String("avatar_id", message.AvatarID.String()),
			slog.Any("err", err),
		)
		if nackErr := delivery.Nack(false, true); nackErr != nil {
			span.recordError(nackErr)
			return nackErr
		}
		return nil
	}
	if ackErr := delivery.Ack(false); ackErr != nil {
		span.recordError(ackErr)
		return ackErr
	}
	return nil
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

// Close закрывает соединение RabbitMQ вместе со связанными каналами чтения.
func (c *Consumer) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.CloseDeadline(time.Now().Add(consumerCloseTimeout))
}
