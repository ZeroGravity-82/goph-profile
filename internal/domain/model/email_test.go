package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalizeEmail проверяет нормализацию email.
func TestNormalizeEmail(t *testing.T) {
	// Arrange
	rawEmail := "  User@Example.COM  "

	// Act
	email, err := NormalizeEmail(rawEmail)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, Email("user@example.com"), email)
}

// TestNormalizeEmail_RejectsInvalidValue проверяет отклонение некорректных email.
func TestNormalizeEmail_RejectsInvalidValue(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want error
	}{
		{name: "empty", raw: " ", want: ErrInvalidEmail},
		{name: "without at", raw: "user.example.com", want: ErrInvalidEmail},
		{name: "empty local", raw: "@example.com", want: ErrInvalidEmail},
		{name: "empty domain", raw: "user@", want: ErrInvalidEmail},
		{name: "spaces", raw: "user name@example.com", want: ErrInvalidEmail},
		{
			name: "too long",
			raw:  strings.Repeat("a", MaxEmailLength-10) + "@example.com",
			want: ErrEmailTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			rawEmail := tt.raw

			// Act
			_, err := NormalizeEmail(rawEmail)

			// Assert
			require.ErrorIs(t, err, tt.want)
		})
	}
}
