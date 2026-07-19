package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

const (
	formFileField                     = "file"
	maxAvatarFileSizeBytes            = 10 * 1024 * 1024
	maxFormMultipartOverheadSizeBytes = 1024 * 1024
)

var errAvatarFileTooLarge = errors.New("avatar file is too large")

// avatarUseCase описывает сценарии работы с аватарками: загрузка аватарки и получение метаданных аватарки.
type avatarUseCase interface {
	UploadAvatar(ctx context.Context, in usecase.UploadAvatarInput) (usecase.UploadAvatarOutput, error)
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

// uploadAvatar парсит multipart-запрос, проверяет X-User-ID и передает файл в сценарий загрузки аватарки.
func (h *AvatarHandler) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserIDHeader(r)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid X-User-ID header", "")
		return
	}

	input, err := parseAvatarUploadRequest(w, r, userID)
	if err != nil {
		h.writeAvatarUploadError(w, r, err)
		return
	}

	output, err := h.avatarUseCase.UploadAvatar(r.Context(), input)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			writeError(h.logger, w, r, http.StatusNotFound, "User not found", "")
			return
		}
		h.writeAvatarUploadError(w, r, err)
		return
	}

	avatarURL, err := avatarURLForID(output.ID)
	if err != nil {
		logError(h.logger, r, "failed to build avatar URL", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	writeJSON(h.logger, w, r, http.StatusCreated, dto.UploadAvatarResponse{
		ID:        output.ID.String(),
		UserID:    output.UserID.String(),
		URL:       avatarURL,
		Status:    string(output.Status),
		CreatedAt: output.CreatedAt,
	})
}

func parseUserIDHeader(r *http.Request) (uuid.UUID, error) {
	userID, err := uuid.Parse(r.Header.Get("X-User-ID"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse X-User-ID header: %w", err)
	}
	if userID == uuid.Nil {
		return uuid.Nil, errors.New("invalid X-User-ID header")
	}
	return userID, nil
}

// parseAvatarUploadRequest парсит HTTP-запрос загрузки аватарки:
// - ограничивает общий размер multipart-тела;
// - достает файл из поля file;
// - читает файл с отдельным лимитом размера;
// - определяет MIME-тип по содержимому;
// - собирает входные данные сценария загрузки аватарки usecase.UploadAvatarInput.
func parseAvatarUploadRequest(
	w http.ResponseWriter,
	r *http.Request,
	userID uuid.UUID,
) (usecase.UploadAvatarInput, error) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxAvatarFileSizeBytes+maxFormMultipartOverheadSizeBytes,
	)

	if err := r.ParseMultipartForm(maxFormMultipartOverheadSizeBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return usecase.UploadAvatarInput{}, errAvatarFileTooLarge
		}
		return usecase.UploadAvatarInput{}, model.ErrInvalidAvatarMetadata
	}

	file, fileHeader, err := r.FormFile(formFileField)
	if err != nil {
		return usecase.UploadAvatarInput{}, model.ErrInvalidAvatarMetadata
	}
	defer func() {
		_ = file.Close()
	}()

	content, err := readAvatarFile(file)
	if err != nil {
		return usecase.UploadAvatarInput{}, err
	}

	mimeType := detectAvatarMIMEType(content)
	if mimeType == "" {
		return usecase.UploadAvatarInput{}, model.ErrInvalidAvatarMetadata
	}

	return usecase.UploadAvatarInput{
		UserID:   userID,
		FileName: fileHeader.Filename,
		MIMEType: mimeType,
		Content:  content,
	}, nil
}

// readAvatarFile читает максимум 10 МиБ + 1 байт: если прочитан лишний байт, значит файл больше разрешенного лимита.
func readAvatarFile(file multipart.File) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(file, maxAvatarFileSizeBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxAvatarFileSizeBytes {
		return nil, errAvatarFileTooLarge
	}
	return content, nil
}

// detectAvatarMIMEType определяет поддерживаемый MIME-тип по содержимому файла: JPEG и PNG распознаются стандартной
// библиотекой, а WebP проверяется отдельно по RIFF-сигнатуре.
func detectAvatarMIMEType(content []byte) string {
	mimeType := http.DetectContentType(content)
	switch mimeType {
	case model.MIMEJPEG, model.MIMEPNG:
		return mimeType
	}
	if isWebP(content) {
		return model.MIMEWebP
	}
	return ""
}

// isWebP проверяет WebP по RIFF-контейнеру: первые 4 байта должны быть "RIFF", байты 8-11 - "WEBP".
func isWebP(content []byte) bool {
	return len(content) >= 12 && string(content[0:4]) == "RIFF" && string(content[8:12]) == "WEBP"
}

// writeAvatarUploadError переводит ошибки файла аватарки в HTTP-ответы: превышение лимита в 413, невалидные
// метаданные в 400, остальные ошибки в 500.
func (h *AvatarHandler) writeAvatarUploadError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errAvatarFileTooLarge) {
		writeErrorWithMaxSize(
			h.logger,
			w,
			r,
			http.StatusRequestEntityTooLarge,
			"File too large",
			maxAvatarFileSizeBytes,
		)
		return
	}
	if errors.Is(err, model.ErrInvalidAvatarMetadata) {
		writeError(
			h.logger,
			w,
			r,
			http.StatusBadRequest,
			"Invalid file format",
			"Supported formats: jpeg, png, webp",
		)
		return
	}
	logError(h.logger, r, "failed to upload avatar file", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}

func avatarURLForID(avatarID uuid.UUID) (string, error) {
	return url.JoinPath(avatarRoutePath, avatarID.String())
}

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
