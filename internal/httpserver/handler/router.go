package handler

import (
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const apiPathPrefix = "/api/v1"

// NewRouter собирает и возвращает HTTP-роутер REST API.
//
// Доступные ручки:
//
//	POST /api/v1/avatars
//	GET /api/v1/avatar?email={email}
//	GET /api/v1/avatars/{avatar_id}
//	GET /api/v1/avatars/{avatar_id}/metadata
//	DELETE /api/v1/avatars/{avatar_id}
//	POST /api/v1/users/resolve
//	PATCH /api/v1/avatar
//	DELETE /api/v1/avatar
func NewRouter(avatarUseCase avatarUseCase, userUseCase userUseCase, logger *slog.Logger) http.Handler {
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
	userHandler := NewUserHandler(userUseCase, logger)

	r.Route(apiPathPrefix, func(r chi.Router) {
		r.With(middleware.AllowContentType("multipart/form-data")).Post("/avatars", avatarHandler.uploadAvatar)
		r.Get("/avatar", avatarHandler.getPublicAvatarByEmail)
		r.Get("/avatars/{avatar_id}", avatarHandler.getAvatar)
		r.Get("/avatars/{avatar_id}/metadata", avatarHandler.getAvatarMetadata)
		r.Delete("/avatars/{avatar_id}", avatarHandler.deleteAvatar)
		r.With(middleware.AllowContentType("application/json")).Post("/users/resolve", userHandler.resolveUserByEmail)
		r.With(middleware.AllowContentType("application/json")).Patch("/avatar", avatarHandler.selectCurrentAvatar)
		r.Delete("/avatar", avatarHandler.deleteCurrentAvatar)
	})

	return r
}
