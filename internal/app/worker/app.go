package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/queue/rabbitmq"
	minioStorage "github.com/ZeroGravity-82/goph-profile/internal/storage/minio"
	"github.com/ZeroGravity-82/goph-profile/internal/storage/postgres"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// App инициализирует зависимости воркера.
type App struct {
	db       *sqlx.DB
	consumer *rabbitmq.Consumer
}

// New создает App: подключается к БД, настраивает хранилища, сценарии и RabbitMQ consumer.
func New(cfg config.WorkerConfig, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	db, err := sqlx.Connect("pgx", cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	app, err := buildApp(context.Background(), cfg, db, logger)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return app, nil
}

func buildApp(ctx context.Context, cfg config.WorkerConfig, db *sqlx.DB, logger *slog.Logger) (*App, error) {
	userRepo, err := postgres.NewUserRepository(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}
	avatarRepo, err := postgres.NewAvatarRepository(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar repository: %w", err)
	}
	transactor, err := postgres.NewTransactor(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}
	fileStorage, err := minioStorage.NewMinIOStorage(
		ctx,
		cfg.FileStorage.Endpoint,
		cfg.FileStorage.AccessKey,
		cfg.FileStorage.SecretKey,
		cfg.FileStorage.Bucket,
		cfg.FileStorage.UseSSL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}

	avatarWorkerUseCase, err := usecase.NewAvatarWorkerUseCase(
		userRepo,
		avatarRepo,
		transactor,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar worker use case: %w", err)
	}

	avatarHandler, err := NewAvatarHandler(avatarWorkerUseCase, fileStorage, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar worker handler: %w", err)
	}

	consumer, err := rabbitmq.NewConsumer(ctx, rabbitmq.Config{
		URL:                        cfg.Queue.URL,
		Exchange:                   cfg.Queue.Exchange,
		AvatarProcessingQueue:      cfg.Queue.AvatarProcessingQueue,
		AvatarDeletionQueue:        cfg.Queue.AvatarDeletionQueue,
		AvatarProcessingRoutingKey: cfg.Queue.AvatarProcessingRoutingKey,
		AvatarDeletionRoutingKey:   cfg.Queue.AvatarDeletionRoutingKey,
	}, avatarHandler, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create rabbitmq consumer: %w", err)
	}

	return &App{db: db, consumer: consumer}, nil
}

// Run запускает чтение задач аватарок из RabbitMQ.
//
// Блокируется до остановки по сигналу завершения или из-за ошибки RabbitMQ.
func (a *App) Run(ctx context.Context) error {
	return a.consumer.Run(ctx)
}

// Close закрывает ресурсы приложения.
func (a *App) Close() error {
	var err error
	if a.consumer != nil {
		err = errors.Join(err, a.consumer.Close())
	}
	if a.db != nil {
		err = errors.Join(err, a.db.Close())
	}
	return err
}
