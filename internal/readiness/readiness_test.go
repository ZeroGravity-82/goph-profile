package readiness

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRun_RunsChecksConcurrently проверяет параллельный запуск проверок готовности.
func TestRun_RunsChecksConcurrently(t *testing.T) {
	// Arrange
	var startedChecks atomic.Int32
	allChecksStarted := make(chan struct{})
	waitForAllChecks := func(ctx context.Context) error {
		if startedChecks.Add(1) == 2 {
			close(allChecksStarted)
		}
		select {
		case <-allChecksStarted:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	checks := Checks{
		"postgres": waitForAllChecks,
		"s3":       waitForAllChecks,
	}

	// Act
	results := Run(context.Background(), checks, time.Second)

	// Assert
	require.NoError(t, results.Err())
	assert.Equal(t, int32(2), startedChecks.Load())
}

// TestRun_AppliesTimeout проверяет ограничение времени выполнения отдельной проверки.
func TestRun_AppliesTimeout(t *testing.T) {
	// Arrange
	checks := Checks{
		"dependency": func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	// Act
	results := Run(context.Background(), checks, time.Millisecond)

	// Assert
	require.ErrorIs(t, results["dependency"], context.DeadlineExceeded)
}

// TestResults_Err_JoinsCheckErrors проверяет объединение ошибок нескольких проверок.
func TestResults_Err_JoinsCheckErrors(t *testing.T) {
	// Arrange
	postgresErr := errors.New("postgres error")
	s3Err := errors.New("s3 error")
	results := Results{
		"postgres": postgresErr,
		"s3":       s3Err,
		"rabbitmq": nil,
	}

	// Act
	err := results.Err()

	// Assert
	require.ErrorIs(t, err, postgresErr)
	require.ErrorIs(t, err, s3Err)
}
