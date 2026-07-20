package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
	"github.com/ZeroGravity-82/goph-profile/web"
)

const (
	formFileField = "file"

	// publicAvatarCacheControl разрешает клиентам кешировать публичную выдачу аватарки на сутки.
	publicAvatarCacheControl = "max-age=86400"

	maxAvatarFileNameLengthBytes      = 255
	maxAvatarFileSizeBytes            = 10 * 1024 * 1024
	maxFormMultipartOverheadSizeBytes = 1024 * 1024
)

var (
	errAvatarFileTooLarge    = errors.New("avatar file is too large")
	errAvatarFileNameInvalid = errors.New("avatar file name is invalid")
	errInvalidAvatarSize     = errors.New("invalid avatar size")
	errInvalidAvatarFormat   = errors.New("invalid avatar format")
	errInvalidUserIDHeader   = errors.New("invalid X-User-ID header")
	errInvalidAvatarID       = errors.New("invalid avatar_id")
	errInvalidRequestBody    = errors.New("invalid request body")
)

var defaultAvatarPNG = web.DefaultAvatarPNG

// avatarUseCase описывает сценарии работы с аватарками: загрузка, выбор и удаление аватарки, публичная выдача аватарки
// и получение метаданных.
type avatarUseCase interface {
	UploadAvatar(ctx context.Context, in usecase.UploadAvatarInput) (usecase.UploadAvatarOutput, error)
	SelectCurrentAvatar(
		ctx context.Context,
		in usecase.SelectCurrentAvatarInput,
	) error
	DeleteCurrentAvatar(ctx context.Context, in usecase.DeleteCurrentAvatarInput) error
	DeleteAvatar(ctx context.Context, in usecase.DeleteAvatarInput) error
	GetCurrentAvatarByEmail(
		ctx context.Context,
		in usecase.GetCurrentAvatarByEmailInput,
	) (usecase.GetCurrentAvatarByEmailOutput, error)
	GetAvatar(ctx context.Context, in usecase.GetAvatarInput) (usecase.GetAvatarOutput, error)
	GetAvatarMetadata(
		ctx context.Context,
		in usecase.GetAvatarMetadataInput,
	) (usecase.GetAvatarMetadataOutput, error)
}

// AvatarHandler обрабатывает HTTP-запросы для аватарок.
type AvatarHandler struct {
	avatarUseCase avatarUseCase
	logger        *slog.Logger
}

// NewAvatarHandler создает новый AvatarHandler.
func NewAvatarHandler(
	avatarUseCase avatarUseCase,
	logger *slog.Logger,
) *AvatarHandler {
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &AvatarHandler{
		avatarUseCase: avatarUseCase,
		logger:        logger,
	}
}
