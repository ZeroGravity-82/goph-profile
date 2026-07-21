package handler

import (
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

func mustRouter(
	t *testing.T,
	avatarUseCase avatarUseCase,
	userUseCase userUseCase,
	logger *slog.Logger,
) http.Handler {
	t.Helper()

	router, err := NewRouter(avatarUseCase, userUseCase, logger)
	require.NoError(t, err)

	return router
}
