package dto

import (
	"time"

	"github.com/google/uuid"
)

// User описывает строку таблицы app_user, представляющую model.User в базе данных.
type User struct {
	ID              uuid.UUID  `db:"id"`
	Email           string     `db:"email"`
	CurrentAvatarID *uuid.UUID `db:"current_avatar_id"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}
