package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/ZeroGravity-82/goph-profile/internal/app/worker/healthserver"
	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/queue/rabbitmq"
	minioStorage "github.com/ZeroGravity-82/goph-profile/internal/storage/minio"
	"github.com/ZeroGravity-82/goph-profile/internal/storage/postgres"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// App инициализирует зависимости воркера.
type App struct {
	db           *pgxpool.Pool
	consumer     *rabbitmq.Consumer
	healthServer *healthserver.Server
}

// New создает App: подключается к БД, настраивает хранилища, сценарии и RabbitMQ-консьюмер.
func New(cfg config.WorkerConfig, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}
	if err = db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	app, err := buildApp(ctx, cfg, db, logger)
	if err != nil {
		db.Close()
		return nil, err
	}
	return app, nil
}

func buildApp(ctx context.Context, cfg config.WorkerConfig, db *pgxpool.Pool, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = logging.NopLogger()
	}

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
	healthServer, err := healthserver.New(
		cfg.HealthServerAddr,
		healthserver.ReadinessChecks{
			"postgres": db.Ping,
			"s3":       fileStorage.Ping,
			"rabbitmq": consumer.Ping,
		},
		logger,
	)
	if err != nil {
		_ = consumer.Close()
		return nil, fmt.Errorf("failed to create worker health server: %w", err)
	}

	return &App{db: db, consumer: consumer, healthServer: healthServer}, nil
}

// Run запускает чтение задач аватарок из RabbitMQ и сервер проверок состояния воркера.
//
// Блокируется до остановки по сигналу завершения, ошибки консьюмера или ошибки сервера проверок состояния.
func (a *App) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return a.consumer.Run(ctx)
	})
	eg.Go(func() error {
		return a.healthServer.Run(ctx)
	})
	return eg.Wait()
}

// Close закрывает ресурсы приложения.
func (a *App) Close() error {
	var err error
	if a.consumer != nil {
		err = errors.Join(err, a.consumer.Close())
	}
	if a.db != nil {
		a.db.Close()
	}
	return err
}
