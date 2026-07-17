package model

import "errors"

var (
	// ErrInvalidID возвращается при пустом доменном идентификаторе.
	ErrInvalidID = errors.New("invalid id")
	// ErrInvalidEmail возвращается при пустом или некорректном email.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrEmailTooLong возвращается при превышении доменного лимита email.
	ErrEmailTooLong = errors.New("email too long")
	// ErrInvalidAvatarMetadata возвращается при некорректных метаданных.
	ErrInvalidAvatarMetadata = errors.New("invalid avatar metadata")
	// ErrFileTooLarge возвращается при слишком большом размере файла.
	ErrFileTooLarge = errors.New("avatar file too large")
	// ErrImageTooLarge возвращается при слишком большом изображении.
	ErrImageTooLarge = errors.New("avatar image too large")
	// ErrInvalidAvatarTransition возвращается при запрещенном переходе статуса.
	ErrInvalidAvatarTransition = errors.New("invalid avatar transition")
	// ErrAvatarForbidden возвращается при операции над чужой аватаркой.
	ErrAvatarForbidden = errors.New("avatar belongs to another user")
	// ErrAvatarNotReady возвращается при неготовой аватарке.
	ErrAvatarNotReady = errors.New("avatar is not ready")
	// ErrAvatarDeleted возвращается при операции над удаленной аватаркой.
	ErrAvatarDeleted = errors.New("avatar is deleted")
)
