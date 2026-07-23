package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/queue/rabbitmq"
	minioStorage "github.com/ZeroGravity-82/goph-profile/internal/storage/minio"
	"github.com/ZeroGravity-82/goph-profile/internal/storage/postgres"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// App инициализирует зависимости сервиса.
type App struct {
	db        *pgxpool.Pool
	publisher *rabbitmq.Publisher
	httpSrv   *httpserver.HTTPServer
	logger    *slog.Logger
}

// New создает App: подключается к БД, настраивает хранилища, сценарии и HTTP-сервер.
func New(cfg config.ServerConfig, logger *slog.Logger) (*App, error) {
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

func buildApp(ctx context.Context, cfg config.ServerConfig, db *pgxpool.Pool, logger *slog.Logger) (*App, error) {
	tlsCert, err := tls.LoadX509KeyPair(cfg.TLSCertPath, cfg.TLSKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tls certificate: %w", err)
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

	userUseCase, err := usecase.NewUserUseCase(userRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create user use case: %w", err)
	}

	publisher, err := rabbitmq.NewPublisher(ctx, rabbitmq.Config{
		URL:                        cfg.Queue.URL,
		Exchange:                   cfg.Queue.Exchange,
		AvatarProcessingQueue:      cfg.Queue.AvatarProcessingQueue,
		AvatarDeletionQueue:        cfg.Queue.AvatarDeletionQueue,
		AvatarProcessingRoutingKey: cfg.Queue.AvatarProcessingRoutingKey,
		AvatarDeletionRoutingKey:   cfg.Queue.AvatarDeletionRoutingKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create rabbitmq publisher: %w", err)
	}
	closePublisherOnError := true
	defer func() {
		if closePublisherOnError {
			_ = publisher.Close()
		}
	}()

	avatarUseCase, err := usecase.NewAvatarUseCase(
		userRepo,
		avatarRepo,
		transactor,
		fileStorage,
		publisher,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar use case: %w", err)
	}

	httpSrv, err := httpserver.NewHTTPServer(
		cfg.HTTPServerAddr,
		httpTLSConfig(tlsCert),
		avatarUseCase,
		userUseCase,
		httpserver.HealthChecks{
			"postgres": db.Ping,
			"s3":       fileStorage.Ping,
			"rabbitmq": publisher.Ping,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http server: %w", err)
	}

	closePublisherOnError = false
	return &App{db: db, publisher: publisher, httpSrv: httpSrv, logger: logger}, nil
}

func httpTLSConfig(tlsCert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}
}

// Run применяет миграции БД и запускает HTTP-сервер.
//
// Блокируется до остановки по сигналу завершения или из-за ошибки HTTP-сервера.
func (a *App) Run(ctx context.Context) error {
	err := a.runMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return a.httpSrv.Run(ctx)
}

// Close закрывает ресурсы приложения.
func (a *App) Close() error {
	var err error
	if a.publisher != nil {
		err = errors.Join(err, a.publisher.Close())
	}
	if a.db != nil {
		a.db.Close()
	}
	return err
}
