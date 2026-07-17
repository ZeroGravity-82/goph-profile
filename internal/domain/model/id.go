package model

// UserID содержит стабильный идентификатор пользователя.
type UserID string

// AvatarID содержит идентификатор аватарки.
type AvatarID string

func validateUserID(id UserID) error {
	if id == "" {
		return ErrInvalidID
	}
	return nil
}

func validateAvatarID(id AvatarID) error {
	if id == "" {
		return ErrInvalidID
	}
	return nil
}
