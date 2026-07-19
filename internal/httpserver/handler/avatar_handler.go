package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

const (
	avatarFormFileField            = "file"
	maxAvatarFormMultipartOverhead = 1024 * 1024
)

var errAvatarFileTooLarge = errors.New("avatar file is too large")

// AvatarUploader описывает сценарий загрузки аватарки.
type AvatarUploader interface {
	UploadAvatar(ctx context.Context, in usecase.UploadAvatarInput) (usecase.UploadAvatarOutput, error)
}

// AvatarHandler обрабатывает HTTP-запросы для аватарок.
type AvatarHandler struct {
	uploader AvatarUploader
	logger   *slog.Logger
}

// NewAvatarHandler создает AvatarHandler.
func NewAvatarHandler(uploader AvatarUploader, logger *slog.Logger) *AvatarHandler {
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &AvatarHandler{uploader: uploader, logger: logger}
}

// uploadAvatar парсит multipart-запрос, проверяет X-User-ID  и передает файл в сценарий загрузки аватарки.
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

	output, err := h.uploader.UploadAvatar(r.Context(), input)
	if err != nil {
		h.writeUploadUseCaseError(w, r, err)
		return
	}

	avatarURL, err := url.JoinPath(avatarRoutePath, output.ID.String())
	if err != nil {
		h.logInternalServerError(r, "failed to build avatar URL", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	writeJSON(h.logger, w, r, http.StatusCreated, uploadAvatarResponse{
		ID:        output.ID.String(),
		UserID:    output.UserID.String(),
		URL:       avatarURL,
		Status:    string(output.Status),
		CreatedAt: output.CreatedAt,
	})
}

type uploadAvatarResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	MaxSize int64  `json:"max_size,omitempty"`
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
		model.MaxAvatarFileSizeBytes+maxAvatarFormMultipartOverhead,
	)

	if err := r.ParseMultipartForm(maxAvatarFormMultipartOverhead); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return usecase.UploadAvatarInput{}, errAvatarFileTooLarge
		}
		return usecase.UploadAvatarInput{}, model.ErrInvalidAvatarMetadata
	}

	file, fileHeader, err := r.FormFile(avatarFormFileField)
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
	content, err := io.ReadAll(io.LimitReader(file, model.MaxAvatarFileSizeBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > model.MaxAvatarFileSizeBytes {
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
	return len(content) >= 12 &&
		string(content[0:4]) == "RIFF" &&
		string(content[8:12]) == "WEBP"
}

// writeAvatarUploadError переводит ошибки файла аватарки в HTTP-ответы: превышение лимита в 413, невалидные
// метаданные в 400, остальные ошибки в 500.
func (h *AvatarHandler) writeAvatarUploadError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errAvatarFileTooLarge) || errors.Is(err, model.ErrFileTooLarge) {
		writeErrorWithMaxSize(h.logger, w, r, http.StatusRequestEntityTooLarge, "File too large")
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
	h.logInternalServerError(r, "failed to upload avatar file", err)
	writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
}

func (h *AvatarHandler) writeUploadUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, usecase.ErrUserNotFound) {
		writeError(h.logger, w, r, http.StatusNotFound, "User not found", "")
		return
	}
	h.writeAvatarUploadError(w, r, err)
}

func (h *AvatarHandler) logInternalServerError(r *http.Request, message string, err error) {
	h.logger.ErrorContext(
		r.Context(),
		message,
		slog.Any("error", err),
		slog.String("method", r.Method),
		slog.String("uri", r.RequestURI),
	)
}

func writeError(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	message string,
	details string,
) {
	writeJSON(logger, w, r, statusCode, errorResponse{
		Error:   message,
		Details: details,
	})
}

func writeErrorWithMaxSize(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	message string,
) {
	writeJSON(logger, w, r, statusCode, errorResponse{
		Error:   message,
		MaxSize: model.MaxAvatarFileSizeBytes,
	})
}

func writeJSON(logger *slog.Logger, w http.ResponseWriter, r *http.Request, statusCode int, response any) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(
			r.Context(),
			"failed to write HTTP response",
			slog.Any("error", err),
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.Int("status", statusCode),
		)
	}
}
