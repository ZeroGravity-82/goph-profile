package dto

import "time"

// UploadAvatarResponse описывает JSON-ответ успешной загрузки аватарки.
type UploadAvatarResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// SelectCurrentAvatarRequest описывает JSON-запрос выбора текущей аватарки пользователя.
type SelectCurrentAvatarRequest struct {
	AvatarID string `json:"avatar_id"`
}

// AvatarThumbnailResponse описывает доступную миниатюру аватарки в JSON-ответе с публичными метаданными аватарки.
type AvatarThumbnailResponse struct {
	Size string `json:"size"`
	URL  string `json:"url"`
}

// AvatarMetadataResponse описывает JSON-ответ с публичными метаданными аватарки.
type AvatarMetadataResponse struct {
	ID         string                    `json:"id"`
	UserID     string                    `json:"user_id"`
	FileName   string                    `json:"file_name"`
	MIMEType   string                    `json:"mime_type"`
	SizeBytes  int64                     `json:"size_bytes"`
	Width      *int                      `json:"width,omitempty"`
	Height     *int                      `json:"height,omitempty"`
	Status     string                    `json:"status"`
	Thumbnails []AvatarThumbnailResponse `json:"thumbnails"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}

// ListUserAvatarsItemResponse описывает аватарку в JSON-ответе со списком аватарок пользователя.
type ListUserAvatarsItemResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	URL       string    `json:"url"`
	FileName  string    `json:"file_name"`
	MIMEType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	Width     *int      `json:"width,omitempty"`
	Height    *int      `json:"height,omitempty"`
	Status    string    `json:"status"`
	IsCurrent bool      `json:"is_current"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListUserAvatarsResponse описывает JSON-ответ со списком аватарок пользователя.
type ListUserAvatarsResponse struct {
	Avatars []ListUserAvatarsItemResponse `json:"avatars"`
}

// ErrorResponse описывает JSON-ответ с ошибкой HTTP API.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	MaxSize int64  `json:"max_size,omitempty"`
}
