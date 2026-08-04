package minio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	minioV7 "github.com/minio/minio-go/v7"
	"github.com/sony/gobreaker/v2"
)

const (
	minioCircuitBreakerFailureThreshold uint32 = 5
	minioCircuitBreakerMaxRequests      uint32 = 1
	minioCircuitBreakerTimeout                 = 30 * time.Second
)

var errMinIOCircuitBreakerOpen = errors.New("minio circuit breaker is open")

// minioCircuitBreaker временно блокирует обращения к недоступному хранилищу MinIO.
type minioCircuitBreaker struct {
	breaker *gobreaker.CircuitBreaker[struct{}]
}

// newMinIOCircuitBreaker создает circuit breaker с заданным порогом ошибок и временем до пробного запроса.
func newMinIOCircuitBreaker(failureThreshold uint32, timeout time.Duration) *minioCircuitBreaker {
	return &minioCircuitBreaker{
		breaker: gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
			Name:        "minio",
			MaxRequests: minioCircuitBreakerMaxRequests,
			Timeout:     timeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= failureThreshold
			},
			IsSuccessful: isMinIOAvailable,
			IsExcluded: func(err error) bool {
				return errors.Is(err, context.Canceled)
			},
		}),
	}
}

// Execute выполняет операцию, если circuit breaker разрешает обращение к MinIO.
func (b *minioCircuitBreaker) Execute(operation func() error) error {
	_, err := b.breaker.Execute(func() (struct{}, error) {
		return struct{}{}, operation()
	})
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return fmt.Errorf("%w: %v", errMinIOCircuitBreakerOpen, err)
	}
	return err
}

// isMinIOAvailable определяет, указывает ли результат запроса на доступность MinIO.
func isMinIOAvailable(err error) bool {
	if err == nil {
		return true
	}
	response := minioV7.ToErrorResponse(err)
	return response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError
}
