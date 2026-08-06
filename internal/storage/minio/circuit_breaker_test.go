package minio

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	minioV7 "github.com/minio/minio-go/v7"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMinIOCircuitBreaker_Execute_RejectsRequestWhenOpen проверяет блокировку запросов после достижения порога ошибок.
func TestMinIOCircuitBreaker_Execute_RejectsRequestWhenOpen(t *testing.T) {
	// Arrange
	breaker := newMinIOCircuitBreaker(2, time.Minute)
	dependencyErr := errors.New("minio is unavailable")
	calls := 0
	operation := func() error {
		calls++
		return dependencyErr
	}
	require.ErrorIs(t, breaker.Execute(operation), dependencyErr)
	require.ErrorIs(t, breaker.Execute(operation), dependencyErr)

	// Act
	err := breaker.Execute(operation)

	// Assert
	require.ErrorIs(t, err, errMinIOCircuitBreakerOpen)
	assert.Equal(t, 2, calls)
}

// TestMinIOCircuitBreaker_Execute_ClosesAfterSuccessfulProbe проверяет восстановление после успешного пробного запроса.
func TestMinIOCircuitBreaker_Execute_ClosesAfterSuccessfulProbe(t *testing.T) {
	// Arrange
	breaker := newMinIOCircuitBreaker(1, time.Millisecond)
	dependencyErr := errors.New("minio is unavailable")
	operationErr := dependencyErr
	calls := 0
	operation := func() error {
		calls++
		return operationErr
	}
	require.ErrorIs(t, breaker.Execute(operation), dependencyErr)
	require.Eventually(t, func() bool {
		return breaker.breaker.State() == gobreaker.StateHalfOpen
	}, time.Second, time.Millisecond)
	operationErr = nil

	// Act
	err := breaker.Execute(operation)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.Equal(t, gobreaker.StateClosed, breaker.breaker.State())
}

// Test_isMinIOAvailable проверяет классификацию ответов MinIO по доступности хранилища.
func Test_isMinIOAvailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "success",
			err:  nil,
			want: true,
		},
		{
			name: "client error",
			err:  minioV7.ErrorResponse{Code: "NoSuchKey", StatusCode: http.StatusNotFound},
			want: true,
		},
		{
			name: "server error",
			err:  minioV7.ErrorResponse{Code: "InternalError", StatusCode: http.StatusServiceUnavailable},
			want: false,
		},
		{
			name: "connection error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := isMinIOAvailable(tt.err)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}
