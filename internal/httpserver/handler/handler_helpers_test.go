package handler

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustAvatarHandler(t *testing.T, avatarUseCase avatarUseCase, logger *slog.Logger) *AvatarHandler {
	t.Helper()

	handler, err := NewAvatarHandler(avatarUseCase, logger)
	require.NoError(t, err)

	return handler
}

func mustUserHandler(t *testing.T, userUseCase userUseCase, logger *slog.Logger) *UserHandler {
	t.Helper()

	handler, err := NewUserHandler(userUseCase, logger)
	require.NoError(t, err)

	return handler
}

func mustHealthHandler(
	t *testing.T,
	checks map[string]func(context.Context) error,
	logger *slog.Logger,
) *HealthHandler {
	t.Helper()

	handler, err := NewHealthHandler(checks, logger)
	require.NoError(t, err)

	return handler
}

func mustRouter(
	t *testing.T,
	avatarUseCase avatarUseCase,
	userUseCase userUseCase,
	healthChecks map[string]func(context.Context) error,
	logger *slog.Logger,
) http.Handler {
	t.Helper()

	router, err := NewRouter(avatarUseCase, userUseCase, healthChecks, logger)
	require.NoError(t, err)

	return router
}

func okHealthChecks() map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"postgres": okHealthCheck,
		"s3":       okHealthCheck,
		"rabbitmq": okHealthCheck,
	}
}

func okHealthCheck(_ context.Context) error {
	return nil
}

func healthCheckError(err error) func(context.Context) error {
	return func(_ context.Context) error {
		return err
	}
}
