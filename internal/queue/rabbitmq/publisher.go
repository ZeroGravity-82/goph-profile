package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

const (
	publishTimeout = 5 * time.Second
)

// Config описывает настройки паблишера RabbitMQ для задач аватарок.
type Config struct {
	URL                        string
	Exchange                   string
	AvatarProcessingQueue      string
	AvatarDeletionQueue        string
	AvatarProcessingRoutingKey string
	AvatarDeletionRoutingKey   string
}

// Publisher публикует задачи аватарок в RabbitMQ.
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     Config
	mu      sync.Mutex
}

// NewPublisher создает подключение к RabbitMQ, включает подтверждения публикации и создает обменник, очереди и связи
// между ними.
func NewPublisher(ctx context.Context, cfg Config) (*Publisher, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to create rabbitmq publisher: %w", ctx.Err())
	default:
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	publisher, err := newPublisherWithConnection(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return publisher, nil
}

// validateConfig проверяет обязательные настройки подключения и очередей RabbitMQ.
func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return errors.New("rabbitmq URL is not provided")
	}
	if strings.TrimSpace(cfg.Exchange) == "" {
		return errors.New("rabbitmq exchange is not provided")
	}
	if strings.TrimSpace(cfg.AvatarProcessingQueue) == "" {
		return errors.New("rabbitmq avatar processing queue is not provided")
	}
	if strings.TrimSpace(cfg.AvatarDeletionQueue) == "" {
		return errors.New("rabbitmq avatar deletion queue is not provided")
	}
	if strings.TrimSpace(cfg.AvatarProcessingRoutingKey) == "" {
		return errors.New("rabbitmq avatar processing routing key is not provided")
	}
	if strings.TrimSpace(cfg.AvatarDeletionRoutingKey) == "" {
		return errors.New("rabbitmq avatar deletion routing key is not provided")
	}
	return nil
}

// newPublisherWithConnection создает канал RabbitMQ, включает подтверждения публикации и готовит очереди.
func newPublisherWithConnection(conn *amqp.Connection, cfg Config) (*Publisher, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open rabbitmq channel: %w", err)
	}

	if err = channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("failed to enable rabbitmq publish confirmations: %w", err)
	}
	publisher := &Publisher{
		conn:    conn,
		channel: channel,
		cfg:     cfg,
	}
	if err = publisher.declareTopology(); err != nil {
		_ = channel.Close()
		return nil, err
	}
	return publisher, nil
}

// declareTopology создает обменник и привязывает к нему очереди задач аватарок.
func (p *Publisher) declareTopology() error {
	if err := p.channel.ExchangeDeclare(p.cfg.Exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare rabbitmq exchange: %w", err)
	}
	if err := p.declareQueueBinding(p.cfg.AvatarProcessingQueue, p.cfg.AvatarProcessingRoutingKey); err != nil {
		return err
	}
	if err := p.declareQueueBinding(p.cfg.AvatarDeletionQueue, p.cfg.AvatarDeletionRoutingKey); err != nil {
		return err
	}
	return nil
}

// declareQueueBinding создает очередь и связывает ее с обменником по ключу маршрутизации.
func (p *Publisher) declareQueueBinding(queueName string, routingKey string) error {
	queue, err := p.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare rabbitmq queue: %w", err)
	}
	if err = p.channel.QueueBind(queue.Name, routingKey, p.cfg.Exchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind rabbitmq queue: %w", err)
	}
	return nil
}

type avatarProcessingMessage struct {
	AvatarID          string `json:"avatar_id"`
	UserID            string `json:"user_id"`
	ObjectKeyOriginal string `json:"object_key_original"`
}

// PublishAvatarProcessing публикует задачу обработки исходного файла аватарки.
func (p *Publisher) PublishAvatarProcessing(ctx context.Context, message usecase.AvatarProcessingMessage) error {
	return p.publishJSON(
		ctx,
		p.cfg.AvatarProcessingRoutingKey,
		avatarProcessingMessageID(message),
		newAvatarProcessingMessage(message),
	)
}

// newAvatarProcessingMessage преобразует задачу обработки аватарки в JSON-сообщение RabbitMQ.
func newAvatarProcessingMessage(message usecase.AvatarProcessingMessage) avatarProcessingMessage {
	return avatarProcessingMessage{
		AvatarID:          message.AvatarID.String(),
		UserID:            message.UserID.String(),
		ObjectKeyOriginal: message.ObjectKeyOriginal,
	}
}

// avatarProcessingMessageID возвращает стабильный идентификатор сообщения обработки аватарки.
func avatarProcessingMessageID(message usecase.AvatarProcessingMessage) string {
	return "avatar-processing:" + message.AvatarID.String()
}

type avatarDeletionMessage struct {
	AvatarID   string   `json:"avatar_id"`
	ObjectKeys []string `json:"object_keys"`
}

// PublishAvatarDeletion публикует задачу удаления файлов аватарки.
func (p *Publisher) PublishAvatarDeletion(ctx context.Context, message usecase.AvatarDeletionMessage) error {
	return p.publishJSON(
		ctx,
		p.cfg.AvatarDeletionRoutingKey,
		avatarDeletionMessageID(message),
		newAvatarDeletionMessage(message),
	)
}

// newAvatarDeletionMessage преобразует задачу удаления файлов аватарки в JSON-сообщение RabbitMQ.
func newAvatarDeletionMessage(message usecase.AvatarDeletionMessage) avatarDeletionMessage {
	return avatarDeletionMessage{
		AvatarID:   message.AvatarID.String(),
		ObjectKeys: message.ObjectKeys,
	}
}

// avatarDeletionMessageID возвращает стабильный идентификатор сообщения удаления файлов аватарки.
func avatarDeletionMessageID(message usecase.AvatarDeletionMessage) string {
	return "avatar-deletion:" + message.AvatarID.String()
}

// publishJSON публикует JSON-сообщение и ждет подтверждения RabbitMQ.
func (p *Publisher) publishJSON(ctx context.Context, routingKey string, messageID string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal rabbitmq message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		p.cfg.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    messageID,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish rabbitmq message: %w", err)
	}
	if confirmation == nil {
		return errors.New("rabbitmq publish confirmation is not available")
	}

	ack, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait rabbitmq publish confirmation: %w", err)
	}
	if !ack {
		return errors.New("rabbitmq publish was not acknowledged")
	}
	return nil
}

// Close закрывает канал и соединение RabbitMQ.
func (p *Publisher) Close() error {
	var err error
	if p.channel != nil {
		err = errors.Join(err, p.channel.Close())
	}
	if p.conn != nil {
		err = errors.Join(err, p.conn.Close())
	}
	return err
}
