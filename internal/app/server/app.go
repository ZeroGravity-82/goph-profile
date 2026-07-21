package server

import (
	"context"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/ZeroGravity-82/goph-profile/internal/config"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	minioStorage "github.com/ZeroGravity-82/goph-profile/internal/storage/minio"
	"github.com/ZeroGravity-82/goph-profile/internal/storage/postgres"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// App инициализирует зависимости сервиса.
type App struct {
	db      *sqlx.DB
	httpSrv *httpserver.HTTPServer
	logger  *slog.Logger
}

// New создает App: подключается к БД, настраивает хранилища, сценарии и HTTP-сервер.
func New(cfg config.ServerConfig, logger *slog.Logger) (*App, error) {
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

func buildApp(ctx context.Context, cfg config.ServerConfig, db *sqlx.DB, logger *slog.Logger) (*App, error) {
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
	avatarUseCase, err := usecase.NewAvatarUseCase(
		userRepo,
		avatarRepo,
		transactor,
		fileStorage,
		avatarMessagePublisher{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar use case: %w", err)
	}

	httpSrv, err := httpserver.NewHTTPServer(cfg.HTTPServerAddr, avatarUseCase, userUseCase, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create http server: %w", err)
	}

	return &App{db: db, httpSrv: httpSrv, logger: logger}, nil
}

type avatarMessagePublisher struct{}

func (p avatarMessagePublisher) PublishAvatarProcessing(
	_ context.Context,
	_ usecase.AvatarProcessingMessage,
) error {
	return nil
}

func (p avatarMessagePublisher) PublishAvatarDeletion(
	_ context.Context,
	_ usecase.AvatarDeletionMessage,
) error {
	return nil
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
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
