package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// UserRepository реализует доступ к данным пользователей в PostgreSQL.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository создает UserRepository на основе подключения к БД.
func NewUserRepository(db *pgxpool.Pool) (*UserRepository, error) {
	if db == nil {
		return nil, errors.New("postgres database is not provided")
	}
	return &UserRepository{db: db}, nil
}

// GetByID возвращает пользователя по ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	const q = `
SELECT id, email, current_avatar_id, created_at, updated_at
FROM app_user
WHERE id = $1`

	return r.getByID(ctx, id, q)
}

// GetByIDForUpdate возвращает пользователя по ID с блокировкой для обновления записи.
func (r *UserRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (model.User, error) {
	const q = `
SELECT id, email, current_avatar_id, created_at, updated_at
FROM app_user
WHERE id = $1
FOR UPDATE`

	return r.getByID(ctx, id, q)
}

func (r *UserRepository) getByID(ctx context.Context, id uuid.UUID, query string) (model.User, error) {
	exec := executorFromContext(ctx, r.db)
	user, err := scanUser(exec.QueryRow(ctx, query, id))
	if err == nil {
		return user, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, repository.ErrUserNotFound
	}
	if errors.Is(err, model.ErrInvalidEmail) {
		return model.User{}, fmt.Errorf("failed to map user by id: %w", err)
	}
	return model.User{}, fmt.Errorf("failed to select user by id: %w", err)
}

func scanUser(row rowScanner) (model.User, error) {
	var user model.User
	var rawEmail string
	err := row.Scan(
		&user.ID,
		&rawEmail,
		&user.CurrentAvatarID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return model.User{}, err
	}
	email, err := model.NewEmail(rawEmail)
	if err != nil {
		return model.User{}, err
	}
	user.Email = email
	return user, nil
}

// GetByEmail возвращает пользователя по нормализованному email.
func (r *UserRepository) GetByEmail(ctx context.Context, email model.Email) (model.User, error) {
	const q = `
SELECT id, email, current_avatar_id, created_at, updated_at
FROM app_user
WHERE email = $1`

	exec := executorFromContext(ctx, r.db)
	user, err := scanUser(exec.QueryRow(ctx, q, string(email)))
	if err == nil {
		return user, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, repository.ErrUserNotFound
	}
	if errors.Is(err, model.ErrInvalidEmail) {
		return model.User{}, fmt.Errorf("failed to map user by email: %w", err)
	}
	return model.User{}, fmt.Errorf("failed to select user by email: %w", err)
}

// Create создает пользователя с нормализованным email.
func (r *UserRepository) Create(ctx context.Context, email model.Email) (model.User, error) {
	id, err := model.NewUserID()
	if err != nil {
		return model.User{}, fmt.Errorf("failed to create user id: %w", err)
	}
	now := time.Now().UTC()
	user, err := model.NewUser(id, email, now)
	if err != nil {
		return model.User{}, err
	}

	const q = `
INSERT INTO app_user (id, email, current_avatar_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)`

	exec := executorFromContext(ctx, r.db)
	_, err = exec.Exec(
		ctx,
		q,
		user.ID,
		string(user.Email),
		user.CurrentAvatarID,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, repository.ErrEmailAlreadyTaken
		}
		return model.User{}, fmt.Errorf("failed to insert user: %w", err)
	}
	return user, nil
}

// Update сохраняет изменяемые поля пользователя.
func (r *UserRepository) Update(ctx context.Context, user model.User) error {
	const q = `
UPDATE app_user
SET email = $2,
    current_avatar_id = $3,
    updated_at = $4
WHERE id = $1`

	exec := executorFromContext(ctx, r.db)
	result, err := exec.Exec(
		ctx,
		q,
		user.ID,
		string(user.Email),
		user.CurrentAvatarID,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.ErrEmailAlreadyTaken
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return repository.ErrUserNotFound
	}
	return nil
}
