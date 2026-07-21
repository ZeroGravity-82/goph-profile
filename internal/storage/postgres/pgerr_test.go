package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

type unwrapErr struct {
	err error
}

func (e unwrapErr) Error() string {
	return "wrapped: " + e.err.Error()
}

func (e unwrapErr) Unwrap() error {
	return e.err
}

// Test_isUniqueViolation проверяет, что isUniqueViolation отличает нарушение уникальности от остальных ошибок.
func Test_isUniqueViolation(t *testing.T) {
	// Arrange
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "typed unique violation",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			want: true,
		},
		{
			name: "wrapped typed pg error",
			err:  fmt.Errorf("failed to insert user: %w", &pgconn.PgError{Code: pgerrcode.UniqueViolation}),
			want: true,
		},
		{
			name: "custom unwrap chain",
			err:  unwrapErr{err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}},
			want: true,
		},
		{
			name: "fallback by text",
			err:  errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"),
			want: true,
		},
		{
			name: "typed other pg error",
			err:  &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation},
			want: false,
		},
		{
			name: "other error text",
			err:  errors.New("plain error"),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := isUniqueViolation(tt.err)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}
