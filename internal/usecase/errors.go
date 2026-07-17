package usecase

import "errors"

var (
	// ErrInvalidEmail возвращается при некорректном email.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrUserNotFound возвращается при отсутствии пользователя.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailAlreadyTaken возвращается при конфликте уникальности email.
	ErrEmailAlreadyTaken = errors.New("email already taken")
)
