package handler

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// getAvatarMetadata парсит avatar_id из пути и возвращает публичные метаданные аватарки в JSON-ответе.
func (h *AvatarHandler) getAvatarMetadata(w http.ResponseWriter, r *http.Request) {
	avatarID, err := parseAvatarIDPathParam(r)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid avatar_id", "")
		return
	}

	output, err := h.avatarUseCase.GetAvatarMetadata(r.Context(), usecase.GetAvatarMetadataInput{
		AvatarID: avatarID,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrAvatarNotFound) {
			writeError(h.logger, w, r, http.StatusNotFound, "Avatar not found", "")
			return
		}
		logError(h.logger, r, "failed to get avatar metadata", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	response, err := newAvatarMetadataResponse(output)
	if err != nil {
		logError(h.logger, r, "failed to build avatar metadata response", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	writeJSON(h.logger, w, r, http.StatusOK, response)
}

func newAvatarMetadataResponse(output usecase.GetAvatarMetadataOutput) (dto.AvatarMetadataResponse, error) {
	thumbnails, err := avatarThumbnailResponses(output)
	if err != nil {
		return dto.AvatarMetadataResponse{}, err
	}

	return dto.AvatarMetadataResponse{
		ID:         output.ID.String(),
		UserID:     output.UserID.String(),
		FileName:   output.FileName,
		MIMEType:   output.MIMEType,
		SizeBytes:  output.SizeBytes,
		Width:      output.Width,
		Height:     output.Height,
		Status:     string(output.Status),
		Thumbnails: thumbnails,
		CreatedAt:  output.CreatedAt,
		UpdatedAt:  output.UpdatedAt,
	}, nil
}

func avatarThumbnailResponses(output usecase.GetAvatarMetadataOutput) ([]dto.AvatarThumbnailResponse, error) {
	thumbnails := make([]dto.AvatarThumbnailResponse, 0, 2)
	if output.ObjectKeyThumb100 != nil {
		thumbnail, err := avatarThumbnailResponseForSize(output.ID, "100x100")
		if err != nil {
			return nil, err
		}
		thumbnails = append(thumbnails, thumbnail)
	}
	if output.ObjectKeyThumb300 != nil {
		thumbnail, err := avatarThumbnailResponseForSize(output.ID, "300x300")
		if err != nil {
			return nil, err
		}
		thumbnails = append(thumbnails, thumbnail)
	}
	return thumbnails, nil
}

func avatarThumbnailResponseForSize(avatarID uuid.UUID, size string) (dto.AvatarThumbnailResponse, error) {
	thumbnailURL, err := avatarURLForID(avatarID)
	if err != nil {
		return dto.AvatarThumbnailResponse{}, err
	}
	values := url.Values{}
	values.Set("size", size)

	return dto.AvatarThumbnailResponse{
		Size: size,
		URL:  thumbnailURL + "?" + values.Encode(),
	}, nil
}
