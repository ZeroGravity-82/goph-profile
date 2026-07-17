package model

import (
	"time"

	"github.com/google/uuid"
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
type Email string

// NewUserID создает UUIDv7 для пользователя.
func NewUserID() (uuid.UUID, error) {
	return uuid.NewV7()
}

// NewUser создает пользователя.
func NewUser(id uuid.UUID, email Email, now time.Time) (User, error) {
	if id == uuid.Nil {
		return User{}, ErrInvalidID
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
