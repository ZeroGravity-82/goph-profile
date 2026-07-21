package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/storage/postgres/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// AvatarRepository реализует доступ к аватаркам в PostgreSQL.
type AvatarRepository struct {
	db *sqlx.DB
}

// NewAvatarRepository создает AvatarRepository на основе подключения к БД.
func NewAvatarRepository(db *sqlx.DB) (*AvatarRepository, error) {
	if db == nil {
		return nil, errors.New("postgres database is not provided")
	}
	return &AvatarRepository{db: db}, nil
}

// GetByID возвращает аватарку по ID.
func (r *AvatarRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Avatar, error) {
	const q = `
SELECT id, user_id, file_name, mime_type, size_bytes, width, height, object_key_original, object_key_thumb_100,
       object_key_thumb_300, status, created_at, updated_at, deleted_at
FROM avatar
WHERE id = $1`

	var row dto.Avatar
	exec := executorFromContext(ctx, r.db)
	if err := exec.GetContext(ctx, &row, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Avatar{}, usecase.ErrAvatarNotFound
		}
		return model.Avatar{}, fmt.Errorf("failed to select avatar by id: %w", err)
	}

	return model.Avatar{
		ID:                row.ID,
		UserID:            row.UserID,
		FileName:          row.FileName,
		MIMEType:          row.MIMEType,
		SizeBytes:         row.SizeBytes,
		Width:             row.Width,
		Height:            row.Height,
		ObjectKeyOriginal: row.ObjectKeyOriginal,
		ObjectKeyThumb100: row.ObjectKeyThumb100,
		ObjectKeyThumb300: row.ObjectKeyThumb300,
		Status:            model.AvatarStatus(row.Status),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		DeletedAt:         row.DeletedAt,
	}, nil
}

// ListByUserID возвращает неудаленные аватарки пользователя.
func (r *AvatarRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Avatar, error) {
	const q = `
SELECT id, user_id, file_name, mime_type, size_bytes, width, height, object_key_original, object_key_thumb_100,
       object_key_thumb_300, status, created_at, updated_at, deleted_at
FROM avatar
WHERE user_id = $1
  AND deleted_at IS NULL
  AND status NOT IN ('deleting', 'deleted')
ORDER BY created_at DESC, id DESC`

	var rows []dto.Avatar
	exec := executorFromContext(ctx, r.db)
	if err := exec.SelectContext(ctx, &rows, q, userID); err != nil {
		return nil, fmt.Errorf("failed to select user avatars: %w", err)
	}

	avatars := make([]model.Avatar, 0, len(rows))
	for _, row := range rows {
		avatars = append(avatars, model.Avatar{
			ID:                row.ID,
			UserID:            row.UserID,
			FileName:          row.FileName,
			MIMEType:          row.MIMEType,
			SizeBytes:         row.SizeBytes,
			Width:             row.Width,
			Height:            row.Height,
			ObjectKeyOriginal: row.ObjectKeyOriginal,
			ObjectKeyThumb100: row.ObjectKeyThumb100,
			ObjectKeyThumb300: row.ObjectKeyThumb300,
			Status:            model.AvatarStatus(row.Status),
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
			DeletedAt:         row.DeletedAt,
		})
	}
	return avatars, nil
}

// Create создает запись аватарки.
func (r *AvatarRepository) Create(ctx context.Context, avatar model.Avatar) error {
	const q = `
INSERT INTO avatar (
    id, user_id, file_name, mime_type, size_bytes, width, height, object_key_original, object_key_thumb_100,
    object_key_thumb_300, status, created_at, updated_at, deleted_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, q, avatarArgs(avatar)...)
	if err != nil {
		return fmt.Errorf("failed to insert avatar: %w", err)
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
	result, err := exec.ExecContext(
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
		return fmt.Errorf("failed to update avatar: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get updated avatar count: %w", err)
	}
	if rowsAffected == 0 {
		return usecase.ErrAvatarNotFound
	}
	return nil
}

// Delete физически удаляет запись аватарки.
func (r *AvatarRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM avatar WHERE id = $1`

	exec := executorFromContext(ctx, r.db)
	result, err := exec.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("failed to delete avatar: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get deleted avatar count: %w", err)
	}
	if rowsAffected == 0 {
		return usecase.ErrAvatarNotFound
	}
	return nil
}
