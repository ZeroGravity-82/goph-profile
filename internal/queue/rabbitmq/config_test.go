package rabbitmq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// Test_validateConfig проверяет обязательные настройки RabbitMQ.
func Test_validateConfig(t *testing.T) {
	// Arrange
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{name: "valid", cfg: validRabbitMQConfig()},
		{name: "url", cfg: Config{}, wantErr: "rabbitmq URL is not provided"},
		{
			name: "exchange",
			cfg: Config{
				URL: "amqp://guest:guest@localhost:5672/",
			},
			wantErr: "rabbitmq exchange is not provided",
		},
		{
			name: "processing queue",
			cfg: Config{
				URL:      "amqp://guest:guest@localhost:5672/",
				Exchange: "goph-profile.avatar",
			},
			wantErr: "rabbitmq avatar processing queue is not provided",
		},
		{
			name: "deletion queue",
			cfg: Config{
				URL:                   "amqp://guest:guest@localhost:5672/",
				Exchange:              "goph-profile.avatar",
				AvatarProcessingQueue: "goph-profile.avatar.processing",
			},
			wantErr: "rabbitmq avatar deletion queue is not provided",
		},
		{
			name: "processing routing key",
			cfg: Config{
				URL:                   "amqp://guest:guest@localhost:5672/",
				Exchange:              "goph-profile.avatar",
				AvatarProcessingQueue: "goph-profile.avatar.processing",
				AvatarDeletionQueue:   "goph-profile.avatar.deletion",
			},
			wantErr: "rabbitmq avatar processing routing key is not provided",
		},
		{
			name: "deletion routing key",
			cfg: Config{
				URL:                        "amqp://guest:guest@localhost:5672/",
				Exchange:                   "goph-profile.avatar",
				AvatarProcessingQueue:      "goph-profile.avatar.processing",
				AvatarDeletionQueue:        "goph-profile.avatar.deletion",
				AvatarProcessingRoutingKey: "avatar.processing",
			},
			wantErr: "rabbitmq avatar deletion routing key is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := validateConfig(tt.cfg)

			// Assert
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestNewPublisher_ReturnsContextErrorBeforeDial проверяет отмену контекста до подключения к RabbitMQ.
func TestNewPublisher_ReturnsContextErrorBeforeDial(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	publisher, err := NewPublisher(ctx, validRabbitMQConfig())

	// Assert
	require.EqualError(t, err, "failed to create rabbitmq publisher: context canceled")
	assert.Nil(t, publisher)
}

// TestNewConsumer_RejectsNilHandler проверяет обязательность обработчика сообщений.
func TestNewConsumer_RejectsNilHandler(t *testing.T) {
	// Act
	consumer, err := NewConsumer(context.Background(), validRabbitMQConfig(), nil, nil)

	// Assert
	require.EqualError(t, err, "avatar message handler is not provided")
	assert.Nil(t, consumer)
}

// TestNewConsumer_ReturnsContextErrorBeforeDial проверяет отмену контекста до подключения к RabbitMQ.
func TestNewConsumer_ReturnsContextErrorBeforeDial(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	consumer, err := NewConsumer(ctx, validRabbitMQConfig(), avatarMessageHandlerStub{}, nil)

	// Assert
	require.EqualError(t, err, "failed to create rabbitmq consumer: context canceled")
	assert.Nil(t, consumer)
}

// TestPublisher_PingWithoutOpenConnection проверяет ошибку проверки закрытого подключения.
func TestPublisher_PingWithoutOpenConnection(t *testing.T) {
	// Act
	err := (&Publisher{}).Ping(context.Background())

	// Assert
	require.EqualError(t, err, "rabbitmq connection is closed")
}

// TestPublisher_PingCanceledContext проверяет отмену контекста до проверки подключения.
func TestPublisher_PingCanceledContext(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	err := (&Publisher{}).Ping(ctx)

	// Assert
	require.EqualError(t, err, "failed to ping rabbitmq: context canceled")
}

// TestConsumer_Ping проверяет готовность запущенного консьюмера с открытыми подключением и каналами.
func TestConsumer_Ping(t *testing.T) {
	// Arrange
	consumer := &Consumer{
		conn:              &amqp.Connection{},
		processingChannel: newShutdownConsumerChannelFake(),
		deletionChannel:   newShutdownConsumerChannelFake(),
	}
	consumer.ready.Store(true)

	// Act
	err := consumer.Ping(context.Background())

	// Assert
	require.NoError(t, err)
}

// TestConsumer_PingBeforeRun проверяет неготовность консьюмера до запуска чтения сообщений.
func TestConsumer_PingBeforeRun(t *testing.T) {
	// Act
	err := (&Consumer{}).Ping(context.Background())

	// Assert
	require.ErrorIs(t, err, errConsumerNotReady)
}

// TestConsumer_PingCanceledContext проверяет отмену контекста до проверки готовности консьюмера.
func TestConsumer_PingCanceledContext(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	err := (&Consumer{}).Ping(ctx)

	// Assert
	require.EqualError(t, err, "failed to ping rabbitmq consumer: context canceled")
}

func validRabbitMQConfig() Config {
	return Config{
		URL:                        "amqp://guest:guest@localhost:5672/",
		Exchange:                   "goph-profile.avatar",
		AvatarProcessingQueue:      "goph-profile.avatar.processing",
		AvatarDeletionQueue:        "goph-profile.avatar.deletion",
		AvatarProcessingRoutingKey: "avatar.processing",
		AvatarDeletionRoutingKey:   "avatar.deletion",
	}
}

type avatarMessageHandlerStub struct{}

func (avatarMessageHandlerStub) HandleAvatarProcessing(_ context.Context, _ usecase.AvatarProcessingMessage) error {
	return nil
}

func (avatarMessageHandlerStub) HandleAvatarDeletion(_ context.Context, _ usecase.AvatarDeletionMessage) error {
	return nil
}
