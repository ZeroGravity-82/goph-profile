package handler

import (
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const avatarRoutePath = "/api/v1/avatars"
const avatarIDRouteParam = "avatar_id"

// NewRouter собирает и возвращает HTTP-роутер REST API.
//
// Доступные ручки:
//
//	POST /api/v1/avatars
//	GET /api/v1/avatars/{avatar_id}/metadata
func NewRouter(avatarUseCase avatarUseCase, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = logging.NopLogger()
	}

	r := chi.NewRouter()
	r.Use(
		middleware.StripSlashes,
		middleware.RealIP,
		withLogging(logger),
		withGzip(logger),
	)
	avatarHandler := NewAvatarHandler(avatarUseCase, logger)

	r.With(middleware.AllowContentType("multipart/form-data")).Post(avatarRoutePath, avatarHandler.uploadAvatar)
	r.Get(avatarRoutePath+"/{"+avatarIDRouteParam+"}/metadata", avatarHandler.getAvatarMetadata)

	return r
}
