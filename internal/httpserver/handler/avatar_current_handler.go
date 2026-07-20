package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// selectCurrentAvatar парсит X-User-ID и avatar_id из JSON-тела, затем передает их в сценарий выбора текущей аватарки.
func (h *AvatarHandler) selectCurrentAvatar(w http.ResponseWriter, r *http.Request) {
	input, err := parseSelectCurrentAvatarRequest(r)
	if err != nil {
		h.writeSelectCurrentAvatarParseError(w, r, err)
		return
	}

	if err = h.avatarUseCase.SelectCurrentAvatar(r.Context(), input); err != nil {
		h.writeSelectCurrentAvatarUseCaseError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseSelectCurrentAvatarRequest(r *http.Request) (usecase.SelectCurrentAvatarInput, error) {
	headerUserID, err := parseUserIDHeader(r)
	if err != nil {
		return usecase.SelectCurrentAvatarInput{}, errInvalidUserIDHeader
	}

	request, err := selectCurrentAvatarRequestFromRequest(r)
	if err != nil {
		return usecase.SelectCurrentAvatarInput{}, errInvalidRequestBody
	}
	avatarID, err := parseAvatarID(request.AvatarID)
	if err != nil {
		return usecase.SelectCurrentAvatarInput{}, errInvalidAvatarID
	}

	return usecase.SelectCurrentAvatarInput{
		UserID:   headerUserID,
		AvatarID: avatarID,
	}, nil
}

func selectCurrentAvatarRequestFromRequest(r *http.Request) (dto.SelectCurrentAvatarRequest, error) {
	var request dto.SelectCurrentAvatarRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return dto.SelectCurrentAvatarRequest{}, err
	}
	return request, nil
}

func parseAvatarID(raw string) (uuid.UUID, error) {
	avatarID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse avatar_id: %w", err)
	}
	if avatarID == uuid.Nil {
		return uuid.Nil, errors.New("invalid avatar_id")
	}
	return avatarID, nil
}

func (h *AvatarHandler) writeSelectCurrentAvatarParseError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errInvalidUserIDHeader) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid X-User-ID header", "")
		return
	}
	if errors.Is(err, errInvalidRequestBody) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid request body", "")
		return
	}
	if errors.Is(err, errInvalidAvatarID) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid avatar_id", "")
		return
	}
	if errors.Is(err, model.ErrAvatarForbidden) {
		writeError(h.logger, w, r, http.StatusForbidden, "Forbidden", "")
		return
	}
	logError(h.logger, r, "failed to parse select current avatar request", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}

func (h *AvatarHandler) writeSelectCurrentAvatarUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
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
	if errors.Is(err, model.ErrAvatarNotReady) {
		writeError(h.logger, w, r, http.StatusConflict, "Avatar is not ready", "")
		return
	}
	if errors.Is(err, model.ErrAvatarDeleted) {
		writeError(h.logger, w, r, http.StatusConflict, "Avatar is deleted", "")
		return
	}
	logError(h.logger, r, "failed to select current avatar", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}

// deleteCurrentAvatar парсит X-User-ID и передает пользователя в сценарий удаления текущей аватарки.
func (h *AvatarHandler) deleteCurrentAvatar(w http.ResponseWriter, r *http.Request) {
	input, err := parseDeleteCurrentAvatarRequest(r)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid X-User-ID header", "")
		return
	}

	if err = h.avatarUseCase.DeleteCurrentAvatar(r.Context(), input); err != nil {
		h.writeDeleteCurrentAvatarUseCaseError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseDeleteCurrentAvatarRequest(r *http.Request) (usecase.DeleteCurrentAvatarInput, error) {
	userID, err := parseUserIDHeader(r)
	if err != nil {
		return usecase.DeleteCurrentAvatarInput{}, errInvalidUserIDHeader
	}

	return usecase.DeleteCurrentAvatarInput{UserID: userID}, nil
}

func (h *AvatarHandler) writeDeleteCurrentAvatarUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
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
	logError(h.logger, r, "failed to delete current avatar", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}
