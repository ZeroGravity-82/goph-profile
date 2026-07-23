package repository

import "errors"

var (
	// ErrUserNotFound возвращается репозиторием, когда пользователь не найден.
	ErrUserNotFound = errors.New("user not found")
	// ErrAvatarNotFound возвращается репозиторием, когда аватарка не найдена.
	ErrAvatarNotFound = errors.New("avatar not found")
	// ErrEmailAlreadyTaken возвращается репозиторием при конфликте уникальности email.
	ErrEmailAlreadyTaken = errors.New("email already taken")
)
