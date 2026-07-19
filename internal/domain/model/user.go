package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxEmailSizeBytes ограничивает нормализованный email пользователя.
	MaxEmailSizeBytes = 255
)

// User описывает пользователя сервиса.
type User struct {
	ID              uuid.UUID
	Email           Email
	CurrentAvatarID *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Email содержит нормализованный email пользователя.
//
// Новый email нужно создавать через NewEmail, только так он буден валидным и нормализованным.
type Email string

// NewEmail нормализует email и проверяет доменные ограничения.
func NewEmail(raw string) (Email, error) {
	email := Email(strings.ToLower(strings.TrimSpace(raw)))
	if err := email.Validate(); err != nil {
		return "", err
	}
	return email, nil
}

// Validate проверяет доменные ограничения нормализованного email.
func (e Email) Validate() error {
	email := string(e)
	if email == "" {
		return ErrInvalidEmail
	}
	if len(email) > MaxEmailSizeBytes {
		return ErrInvalidEmail
	}
	if strings.Count(email, "@") != 1 {
		return ErrInvalidEmail
	}
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" || strings.ContainsAny(email, " \t\r\n") {
		return ErrInvalidEmail
	}
	if email != strings.ToLower(strings.TrimSpace(email)) {
		return ErrInvalidEmail
	}
	return nil
}

// NewUserID создает UUIDv7 для пользователя.
func NewUserID() (uuid.UUID, error) {
	return uuid.NewV7()
}

// NewUser создает пользователя.
func NewUser(id uuid.UUID, email Email, now time.Time) (User, error) {
	if id == uuid.Nil {
		return User{}, ErrInvalidID
	}
	if err := email.Validate(); err != nil {
		return User{}, err
	}
	return User{ID: id, Email: email, CreatedAt: now, UpdatedAt: now}, nil
}

// SelectCurrentAvatar выбирает готовую аватарку пользователя как текущую.
func (u *User) SelectCurrentAvatar(avatar Avatar, now time.Time) error {
	if avatar.UserID != u.ID {
		return ErrAvatarForbidden
	}
	if err := avatar.CanBeCurrent(); err != nil {
		return err
	}
	if u.CurrentAvatarID != nil && *u.CurrentAvatarID == avatar.ID {
		return nil
	}
	avatarID := avatar.ID
	u.CurrentAvatarID = &avatarID
	u.UpdatedAt = now
	return nil
}

// ClearCurrentAvatar сбрасывает текущую аватарку.
func (u *User) ClearCurrentAvatar(avatarID uuid.UUID, now time.Time) bool {
	if u.CurrentAvatarID == nil || *u.CurrentAvatarID != avatarID {
		return false
	}
	u.CurrentAvatarID = nil
	u.UpdatedAt = now
	return true
}
