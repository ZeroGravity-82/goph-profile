package handler

import (
	"errors"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// deleteAvatar парсит X-User-ID и avatar_id из пути, затем передает их в сценарий удаления конкретной аватарки.
func (h *AvatarHandler) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	input, err := parseDeleteAvatarRequest(r)
	if err != nil {
		h.writeDeleteAvatarParseError(w, r, err)
		return
	}

	if err = h.avatarUseCase.DeleteAvatar(r.Context(), input); err != nil {
		h.writeDeleteAvatarUseCaseError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseDeleteAvatarRequest(r *http.Request) (usecase.DeleteAvatarInput, error) {
	userID, err := parseUserIDHeader(r)
	if err != nil {
		return usecase.DeleteAvatarInput{}, errInvalidUserIDHeader
	}

	avatarID, err := parseAvatarIDPathParam(r)
	if err != nil {
		return usecase.DeleteAvatarInput{}, errInvalidAvatarID
	}

	return usecase.DeleteAvatarInput{
		UserID:   userID,
		AvatarID: avatarID,
	}, nil
}

func (h *AvatarHandler) writeDeleteAvatarParseError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errInvalidUserIDHeader) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid X-User-ID header", "")
		return
	}
	if errors.Is(err, errInvalidAvatarID) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid avatar_id", "")
		return
	}
	logError(h.logger, r, "failed to parse delete avatar request", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}

func (h *AvatarHandler) writeDeleteAvatarUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, usecase.ErrUserNotFound) {
		writeError(h.logger, w, r, http.StatusNotFound, "User not found", "")
		return
	}
	if errors.Is(err, usecase.ErrAvatarNotFound) {
		writeError(h.logger, w, r, http.StatusNotFound, "Avatar not found", "")
		return
	}
	if errors.Is(err, model.ErrAvatarForbidden) {
		writeError(h.logger, w, r, http.StatusForbidden, "Forbidden", "")
		return
	}
	logError(h.logger, r, "failed to delete avatar", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}
