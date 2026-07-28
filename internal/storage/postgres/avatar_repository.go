package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
)

// AvatarRepository реализует доступ к аватаркам в PostgreSQL.
type AvatarRepository struct {
	db *pgxpool.Pool
}

// NewAvatarRepository создает AvatarRepository на основе подключения к БД.
func NewAvatarRepository(db *pgxpool.Pool) (*AvatarRepository, error) {
	if db == nil {
		return nil, errors.New("postgres database is not provided")
	}
	return &AvatarRepository{db: db}, nil
}

// GetByID возвращает аватарку по ID.
func (r *AvatarRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Avatar, error) {
	ctx, span := startSpan(ctx, "postgres.avatar.get_by_id", "SELECT", "avatar")
	defer span.End()

	const q = `
SELECT id, user_id, file_name, mime_type, size_bytes, width, height, object_key_original, object_key_thumb_100,
       object_key_thumb_300, status, created_at, updated_at, deleted_at
FROM avatar
WHERE id = $1`

	avatar, err := r.getByID(ctx, id, q)
	if err != nil {
		recordSpanError(span, err)
		return model.Avatar{}, err
	}
	return avatar, nil
}

// GetByIDForUpdate возвращает аватарку по ID с блокировкой для обновления записи.
func (r *AvatarRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (model.Avatar, error) {
	ctx, span := startSpan(ctx, "postgres.avatar.get_by_id_for_update", "SELECT FOR UPDATE", "avatar")
	defer span.End()

	const q = `
SELECT id, user_id, file_name, mime_type, size_bytes, width, height, object_key_original, object_key_thumb_100,
       object_key_thumb_300, status, created_at, updated_at, deleted_at
FROM avatar
WHERE id = $1
FOR UPDATE`

	avatar, err := r.getByID(ctx, id, q)
	if err != nil {
		recordSpanError(span, err)
		return model.Avatar{}, err
	}
	return avatar, nil
}

func (r *AvatarRepository) getByID(ctx context.Context, id uuid.UUID, query string) (model.Avatar, error) {
	exec := executorFromContext(ctx, r.db)
	avatar, err := scanAvatar(exec.QueryRow(ctx, query, id))
	if err == nil {
		return avatar, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Avatar{}, repository.ErrAvatarNotFound
	}
	return model.Avatar{}, fmt.Errorf("failed to select avatar by id: %w", err)
}

func scanAvatar(row rowScanner) (model.Avatar, error) {
	var avatar model.Avatar
	var status string
	err := row.Scan(
		&avatar.ID,
		&avatar.UserID,
		&avatar.FileName,
		&avatar.MIMEType,
		&avatar.SizeBytes,
		&avatar.Width,
		&avatar.Height,
		&avatar.ObjectKeyOriginal,
		&avatar.ObjectKeyThumb100,
		&avatar.ObjectKeyThumb300,
		&status,
		&avatar.CreatedAt,
		&avatar.UpdatedAt,
		&avatar.DeletedAt,
	)
	if err != nil {
		return model.Avatar{}, err
	}
	avatar.Status = model.AvatarStatus(status)
	return avatar, nil
}

// ListByUserID возвращает аватарки пользователя.
func (r *AvatarRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Avatar, error) {
	ctx, span := startSpan(ctx, "postgres.avatar.list_by_user_id", "SELECT", "avatar")
	defer span.End()

	const q = `
SELECT id, user_id, file_name, mime_type, size_bytes, width, height, object_key_original, object_key_thumb_100,
       object_key_thumb_300, status, created_at, updated_at, deleted_at
FROM avatar
WHERE user_id = $1
  AND deleted_at IS NULL
  AND status NOT IN ('deleting', 'deleted')
ORDER BY created_at DESC, id DESC`

	exec := executorFromContext(ctx, r.db)
	rows, err := exec.Query(ctx, q, userID)
	if err != nil {
		err = fmt.Errorf("failed to select user avatars: %w", err)
		recordSpanError(span, err)
		return nil, err
	}
	defer rows.Close()

	avatars := make([]model.Avatar, 0)
	for rows.Next() {
		avatar, err := scanAvatar(rows)
		if err != nil {
			err = fmt.Errorf("failed to scan user avatar: %w", err)
			recordSpanError(span, err)
			return nil, err
		}
		avatars = append(avatars, avatar)
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("failed to read user avatars: %w", err)
		recordSpanError(span, err)
		return nil, err
	}
	return avatars, nil
}

// Create создает запись аватарки.
func (r *AvatarRepository) Create(ctx context.Context, avatar model.Avatar) error {
	ctx, span := startSpan(ctx, "postgres.avatar.create", "INSERT", "avatar")
	defer span.End()

	const q = `
INSERT INTO avatar (
    id, user_id, file_name, mime_type, size_bytes, width, height, object_key_original, object_key_thumb_100,
    object_key_thumb_300, status, created_at, updated_at, deleted_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	exec := executorFromContext(ctx, r.db)
	_, err := exec.Exec(ctx, q, avatarArgs(avatar)...)
	if err != nil {
		err = fmt.Errorf("failed to insert avatar: %w", err)
		recordSpanError(span, err)
		return err
	}
	return nil
}

func avatarArgs(avatar model.Avatar) []any {
	return []any{
		avatar.ID,
		avatar.UserID,
		avatar.FileName,
		avatar.MIMEType,
		avatar.SizeBytes,
		avatar.Width,
		avatar.Height,
		avatar.ObjectKeyOriginal,
		avatar.ObjectKeyThumb100,
		avatar.ObjectKeyThumb300,
		string(avatar.Status),
		avatar.CreatedAt,
		avatar.UpdatedAt,
		avatar.DeletedAt,
	}
}

// Update сохраняет изменяемые поля аватарки.
func (r *AvatarRepository) Update(ctx context.Context, avatar model.Avatar) error {
	ctx, span := startSpan(ctx, "postgres.avatar.update", "UPDATE", "avatar")
	defer span.End()

	const q = `
UPDATE avatar
SET file_name = $2,
    mime_type = $3,
    size_bytes = $4,
    width = $5,
    height = $6,
    object_key_original = $7,
    object_key_thumb_100 = $8,
    object_key_thumb_300 = $9,
    status = $10,
    updated_at = $11,
    deleted_at = $12
WHERE id = $1`

	exec := executorFromContext(ctx, r.db)
	result, err := exec.Exec(
		ctx,
		q,
		avatar.ID,
		avatar.FileName,
		avatar.MIMEType,
		avatar.SizeBytes,
		avatar.Width,
		avatar.Height,
		avatar.ObjectKeyOriginal,
		avatar.ObjectKeyThumb100,
		avatar.ObjectKeyThumb300,
		string(avatar.Status),
		avatar.UpdatedAt,
		avatar.DeletedAt,
	)
	if err != nil {
		err = fmt.Errorf("failed to update avatar: %w", err)
		recordSpanError(span, err)
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		recordSpanError(span, repository.ErrAvatarNotFound)
		return repository.ErrAvatarNotFound
	}
	return nil
}

// Delete физически удаляет запись аватарки.
func (r *AvatarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := startSpan(ctx, "postgres.avatar.delete", "DELETE", "avatar")
	defer span.End()

	const q = `DELETE FROM avatar WHERE id = $1`

	exec := executorFromContext(ctx, r.db)
	result, err := exec.Exec(ctx, q, id)
	if err != nil {
		err = fmt.Errorf("failed to delete avatar: %w", err)
		recordSpanError(span, err)
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		recordSpanError(span, repository.ErrAvatarNotFound)
		return repository.ErrAvatarNotFound
	}
	return nil
}
