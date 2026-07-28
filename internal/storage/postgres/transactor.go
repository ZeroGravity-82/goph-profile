package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey string

const txContextKey ctxKey = "tx"

type queryExecutor interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

type rowScanner interface {
	Scan(dest ...any) error
}

// Transactor управляет транзакциями PostgreSQL.
type Transactor struct {
	db *pgxpool.Pool
}

// NewTransactor создает Transactor на основе подключения к БД.
func NewTransactor(db *pgxpool.Pool) (*Transactor, error) {
	if db == nil {
		return nil, errors.New("postgres database is not provided")
	}
	return &Transactor{db: db}, nil
}

// WithinTransaction выполняет fn внутри транзакции.
//
// Если в context уже есть активная транзакция, новая транзакция не открывается, а fn выполняется в существующей.
func (t *Transactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if txFromContext(ctx) != nil {
		return fn(ctx)
	}

	ctx, span := startSpan(ctx, "postgres.transaction", "TRANSACTION", "")
	defer span.End()

	tx, err := t.db.Begin(ctx)
	if err != nil {
		err = fmt.Errorf("failed to begin transaction: %w", err)
		recordSpanError(span, err)
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = fn(context.WithValue(ctx, txContextKey, tx)); err != nil {
		recordSpanError(span, err)
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		err = fmt.Errorf("failed to commit transaction: %w", err)
		recordSpanError(span, err)
		return err
	}
	committed = true
	return nil
}

func executorFromContext(ctx context.Context, db *pgxpool.Pool) queryExecutor {
	if tx := txFromContext(ctx); tx != nil {
		return tx
	}
	return db
}

func txFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txContextKey).(pgx.Tx)
	return tx
}
