package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
	"github.com/ZeroGravity-82/goph-profile/internal/storage/postgres/dto"
)

// UserRepository реализует доступ к данным пользователей в PostgreSQL.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository создает UserRepository на основе подключения к БД.
func NewUserRepository(db *sqlx.DB) (*UserRepository, error) {
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
	var row dto.User
	exec := executorFromContext(ctx, r.db)
	if err := exec.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, repository.ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("failed to select user by id: %w", err)
	}

	user, err := userFromDTO(row)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to map user by id: %w", err)
	}
	return user, nil
}

func userFromDTO(userRow dto.User) (model.User, error) {
	email, err := model.NewEmail(userRow.Email)
	if err != nil {
		return model.User{}, err
	}
	return model.User{
		ID:              userRow.ID,
		Email:           email,
		CurrentAvatarID: userRow.CurrentAvatarID,
		CreatedAt:       userRow.CreatedAt,
		UpdatedAt:       userRow.UpdatedAt,
	}, nil
}

// GetByEmail возвращает пользователя по нормализованному email.
func (r *UserRepository) GetByEmail(ctx context.Context, email model.Email) (model.User, error) {
	const q = `
SELECT id, email, current_avatar_id, created_at, updated_at
FROM app_user
WHERE email = $1`

	var row dto.User
	exec := executorFromContext(ctx, r.db)
	if err := exec.GetContext(ctx, &row, q, string(email)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, repository.ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("failed to select user by email: %w", err)
	}

	user, err := userFromDTO(row)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to map user by email: %w", err)
	}
	return user, nil
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

	row := userToDTO(user)
	exec := executorFromContext(ctx, r.db)
	_, err = exec.ExecContext(
		ctx,
		q,
		row.ID,
		row.Email,
		row.CurrentAvatarID,
		row.CreatedAt,
		row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, repository.ErrEmailAlreadyTaken
		}
		return model.User{}, fmt.Errorf("failed to insert user: %w", err)
	}
	return user, nil
}

func userToDTO(user model.User) dto.User {
	return dto.User{
		ID:              user.ID,
		Email:           string(user.Email),
		CurrentAvatarID: user.CurrentAvatarID,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}

// Update сохраняет изменяемые поля пользователя.
func (r *UserRepository) Update(ctx context.Context, user model.User) error {
	const q = `
UPDATE app_user
SET email = $2,
    current_avatar_id = $3,
    updated_at = $4
WHERE id = $1`

	row := userToDTO(user)
	exec := executorFromContext(ctx, r.db)
	result, err := exec.ExecContext(
		ctx,
		q,
		row.ID,
		row.Email,
		row.CurrentAvatarID,
		row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.ErrEmailAlreadyTaken
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get updated user count: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrUserNotFound
	}
	return nil
}
