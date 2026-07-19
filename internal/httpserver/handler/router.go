package handler

import (
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const avatarRoutePath = "/api/v1/avatars"

// NewRouter собирает и возвращает HTTP-роутер REST API.
//
// Доступные ручки:
//
//	POST /api/v1/avatars
func NewRouter(uploader AvatarUploader, logger *slog.Logger) http.Handler {
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
	avatarHandler := NewAvatarHandler(uploader, logger)

	r.With(middleware.AllowContentType("multipart/form-data")).Post(avatarRoutePath, avatarHandler.uploadAvatar)

	return r
}
