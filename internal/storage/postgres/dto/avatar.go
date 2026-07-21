package dto

import (
	"time"

	"github.com/google/uuid"
)

// Avatar описывает строку таблицы avatar, представляющую model.Avatar в базе данных.
type Avatar struct {
	ID                uuid.UUID  `db:"id"`
	UserID            uuid.UUID  `db:"user_id"`
	FileName          string     `db:"file_name"`
	MIMEType          string     `db:"mime_type"`
	SizeBytes         int64      `db:"size_bytes"`
	Width             *int       `db:"width"`
	Height            *int       `db:"height"`
	ObjectKeyOriginal string     `db:"object_key_original"`
	ObjectKeyThumb100 *string    `db:"object_key_thumb_100"`
	ObjectKeyThumb300 *string    `db:"object_key_thumb_300"`
	Status            string     `db:"status"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
	DeletedAt         *time.Time `db:"deleted_at"`
}
