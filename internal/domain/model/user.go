package model

import "time"

// User описывает пользователя сервиса.
type User struct {
	ID              UserID
	Email           Email
	CurrentAvatarID *AvatarID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewUser создает пользователя с нормализованным email.
func NewUser(id UserID, email string, now time.Time) (User, error) {
	if err := validateUserID(id); err != nil {
		return User{}, err
	}
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	return User{
		ID:        id,
		Email:     normalizedEmail,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
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
func (u *User) ClearCurrentAvatar(avatarID AvatarID, now time.Time) bool {
	if u.CurrentAvatarID == nil || *u.CurrentAvatarID != avatarID {
		return false
	}
	u.CurrentAvatarID = nil
	u.UpdatedAt = now
	return true
}
