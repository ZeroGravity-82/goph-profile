package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// getAvatar парсит avatar_id и query-параметры size/format, затем возвращает бинарный файл аватарки.
func (h *AvatarHandler) getAvatar(w http.ResponseWriter, r *http.Request) {
	input, err := parseGetAvatarRequest(r)
	if err != nil {
		h.writeGetAvatarError(w, r, err)
		return
	}

	output, err := h.avatarUseCase.GetAvatar(r.Context(), input)
	if err != nil {
		h.writeGetAvatarError(w, r, err)
		return
	}

	writeAvatarContent(h.logger, w, r, output.MIMEType, output.Content)
}

func parseGetAvatarRequest(r *http.Request) (usecase.GetAvatarInput, error) {
	avatarID, err := parseAvatarIDPathParam(r)
	if err != nil {
		return usecase.GetAvatarInput{}, model.ErrInvalidID
	}
	size, err := avatarSizeFromRequest(r)
	if err != nil {
		return usecase.GetAvatarInput{}, err
	}
	mimeType, err := avatarMIMETypeFromRequest(r)
	if err != nil {
		return usecase.GetAvatarInput{}, err
	}

	return usecase.GetAvatarInput{
		AvatarID: avatarID,
		Size:     size,
		MIMEType: mimeType,
	}, nil
}

func parseAvatarIDPathParam(r *http.Request) (uuid.UUID, error) {
	avatarID, err := uuid.Parse(chi.URLParam(r, avatarIDRouteParam))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse avatar_id path param: %w", err)
	}
	if avatarID == uuid.Nil {
		return uuid.Nil, errors.New("invalid avatar_id path param")
	}
	return avatarID, nil
}

func avatarSizeFromRequest(r *http.Request) (usecase.AvatarSize, error) {
	switch r.URL.Query().Get(avatarSizeQueryParam) {
	case "", string(usecase.AvatarSizeOriginal):
		return usecase.AvatarSizeOriginal, nil
	case string(usecase.AvatarSize100):
		return usecase.AvatarSize100, nil
	case string(usecase.AvatarSize300):
		return usecase.AvatarSize300, nil
	default:
		return "", errInvalidAvatarSize
	}
}

func avatarMIMETypeFromRequest(r *http.Request) (string, error) {
	switch r.URL.Query().Get(avatarFormatQueryParam) {
	case "":
		return "", nil
	case "jpeg":
		return model.MIMEJPEG, nil
	case "png":
		return model.MIMEPNG, nil
	case "webp":
		return model.MIMEWebP, nil
	default:
		return "", errInvalidAvatarFormat
	}
}

func (h *AvatarHandler) writeGetAvatarError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, usecase.ErrAvatarNotFound) {
		writeError(h.logger, w, r, http.StatusNotFound, "Avatar not found", "")
		return
	}
	if errors.Is(err, model.ErrInvalidAvatarMetadata) || errors.Is(err, errInvalidAvatarFormat) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid format", "")
		return
	}
	if errors.Is(err, errInvalidAvatarSize) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid size", "")
		return
	}
	if errors.Is(err, model.ErrInvalidID) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid avatar_id", "")
		return
	}
	logError(h.logger, r, "failed to get avatar", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}
