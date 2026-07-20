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
const publicAvatarRoutePath = "/api/v1/avatar"
const userResolveRoutePath = "/api/v1/users/resolve"

// NewRouter собирает и возвращает HTTP-роутер REST API.
//
// Доступные ручки:
//
//	POST /api/v1/avatars
//	GET /api/v1/avatar?email={email}
//	GET /api/v1/avatars/{avatar_id}
//	GET /api/v1/avatars/{avatar_id}/metadata
//	POST /api/v1/users/resolve
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

	r.With(middleware.AllowContentType("multipart/form-data")).Post(avatarRoutePath, avatarHandler.uploadAvatar)
	r.Get(publicAvatarRoutePath, avatarHandler.getPublicAvatarByEmail)
	r.Get(avatarRoutePath+"/{"+avatarIDRouteParam+"}", avatarHandler.getAvatar)
	r.Get(avatarRoutePath+"/{"+avatarIDRouteParam+"}/metadata", avatarHandler.getAvatarMetadata)
	r.With(middleware.AllowContentType("application/json")).Post(userResolveRoutePath, userHandler.resolveUserByEmail)

	return r
}
