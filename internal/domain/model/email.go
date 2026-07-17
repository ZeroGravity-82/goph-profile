package model

import "strings"

const (
	// MaxEmailLength ограничивает нормализованный email в таблице app_user.
	MaxEmailLength = 255
)

// Email содержит нормализованный email пользователя.
type Email string

// NormalizeEmail приводит email к доменному виду trim + lowercase.
func NormalizeEmail(raw string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", ErrInvalidEmail
	}
	if len(normalized) > MaxEmailLength {
		return "", ErrEmailTooLong
	}
	if strings.Count(normalized, "@") != 1 {
		return "", ErrInvalidEmail
	}
	local, domain, ok := strings.Cut(normalized, "@")
	if !ok || local == "" || domain == "" || strings.ContainsAny(normalized, " \t\r\n") {
		return "", ErrInvalidEmail
	}
	return Email(normalized), nil
}
