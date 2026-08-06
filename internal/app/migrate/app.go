package migrate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	migrationfiles "github.com/ZeroGravity-82/goph-profile/migrations"
)

const (
	migrationLockRetryPeriodSeconds  = 5
	migrationLockRetryAttempts       = 60
	migrationUnlockRetryPeriodSecond = 2
	migrationUnlockRetryAttempts     = 30
)

// Migrator управляет подключением к PostgreSQL и применением миграций.
type Migrator struct {
	provider *goose.Provider
	logger   *slog.Logger
}

// New создает Migrator: подключается к PostgreSQL и настраивает goose.
func New(databaseURI string, logger *slog.Logger) (*Migrator, error) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	connConfig, err := pgx.ParseConfig(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URI: %w", err)
	}

	// Goose работает с *sql.DB, поэтому создаем его через адаптер pgx/stdlib.
	migrationDB := stdlib.OpenDB(*connConfig)
	if err = migrationDB.PingContext(context.Background()); err != nil {
		_ = migrationDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	sessionLocker, err := lock.NewPostgresSessionLocker(
		lock.WithLockID(lock.DefaultLockID),
		lock.WithLockTimeout(migrationLockRetryPeriodSeconds, migrationLockRetryAttempts),
		lock.WithUnlockTimeout(migrationUnlockRetryPeriodSecond, migrationUnlockRetryAttempts),
	)
	if err != nil {
		_ = migrationDB.Close()
		return nil, fmt.Errorf("failed to create postgres session locker: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		migrationDB,
		migrationfiles.FS,
		goose.WithSessionLocker(sessionLocker),
	)
	if err != nil {
		_ = migrationDB.Close()
		return nil, fmt.Errorf("failed to create goose provider: %w", err)
	}

	return &Migrator{provider: provider, logger: logger}, nil
}

// Run накатывает встроенные goose-миграции.
//
// Для защиты от конкурентного запуска используется PostgreSQL advisory lock через session locker пакета goose:
// несколько параллельных процессов будут ждать одну и ту же блокировку, поэтому миграции в конкретный момент времени
// накатит только один из процессов.
//
// Реализация session locker в goose: https://github.com/pressly/goose/blob/main/lock/postgres.go
// Документация PostgreSQL по advisory lock:
// https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS
func (m *Migrator) Run(ctx context.Context) error {
	results, err := m.provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	for _, res := range results {
		m.logger.InfoContext(ctx, "applied migration",
			slog.Int64("version", res.Source.Version),
			slog.String("source", res.Source.Path),
			slog.Duration("duration", res.Duration),
		)
	}
	m.logger.InfoContext(ctx, "postgres migrations complete", slog.Int("count", len(results)))

	return nil
}

// Close закрывает ресурсы мигратора.
func (m *Migrator) Close() error {
	if m.provider == nil {
		return nil
	}
	return m.provider.Close()
}
