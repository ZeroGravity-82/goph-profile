package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
	"github.com/ZeroGravity-82/goph-profile/web"
)

const (
	formFileField = "file"

	// avatarCacheControl разрешает клиентам кешировать выдачу аватарки на сутки.
	avatarCacheControl = "max-age=86400"

	maxAvatarFileNameLengthBytes      = 255
	maxAvatarFileSizeBytes            = 10 * 1024 * 1024
	maxFormMultipartOverheadSizeBytes = 1024 * 1024
	maxAvatarImageWidthPixels         = 4096
	maxAvatarImageHeightPixels        = 4096
	maxAvatarImageAreaPixels          = 16_777_216
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

// avatarUseCase описывает сценарии работы с аватарками: загрузка, выбор, удаление, получение списка, выдача аватарки
// по email/ID пользователя и получение метаданных.
type avatarUseCase interface {
	UploadAvatar(ctx context.Context, in usecase.UploadAvatarInput) (usecase.UploadAvatarOutput, error)
	SelectCurrentAvatar(
		ctx context.Context,
		in usecase.SelectCurrentAvatarInput,
	) error
	DeleteCurrentAvatar(ctx context.Context, in usecase.DeleteCurrentAvatarInput) error
	DeleteAvatar(ctx context.Context, in usecase.DeleteAvatarInput) error
	ListUserAvatars(
		ctx context.Context,
		in usecase.ListUserAvatarsInput,
	) (usecase.ListUserAvatarsOutput, error)
	GetCurrentAvatarByEmail(
		ctx context.Context,
		in usecase.GetCurrentAvatarByEmailInput,
	) (usecase.GetCurrentAvatarByEmailOutput, error)
	GetCurrentAvatarByUserID(
		ctx context.Context,
		in usecase.GetCurrentAvatarByUserIDInput,
	) (usecase.GetCurrentAvatarByUserIDOutput, error)
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

// NewAvatarHandler создает AvatarHandler.
func NewAvatarHandler(
	avatarUseCase avatarUseCase,
	logger *slog.Logger,
) (*AvatarHandler, error) {
	if avatarUseCase == nil {
		return nil, errors.New("avatar usecase is not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &AvatarHandler{
		avatarUseCase: avatarUseCase,
		logger:        logger,
	}, nil
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
// - проверяет размеры изображения;
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
	if err = validateAvatarFileName(fileHeader.Filename); err != nil {
		return usecase.UploadAvatarInput{}, err
	}

	content, err := readAvatarFile(file)
	if err != nil {
		return usecase.UploadAvatarInput{}, err
	}

	mimeType := detectAvatarMIMEType(content)
	if mimeType == "" {
		return usecase.UploadAvatarInput{}, model.ErrInvalidAvatarMetadata
	}
	if err = validateAvatarImageDimensions(content); err != nil {
		return usecase.UploadAvatarInput{}, err
	}

	return usecase.UploadAvatarInput{
		UserID:   userID,
		FileName: fileHeader.Filename,
		MIMEType: mimeType,
		Content:  content,
	}, nil
}

func validateAvatarFileName(fileName string) error {
	if fileName == "" || len(fileName) > maxAvatarFileNameLengthBytes {
		return errAvatarFileNameInvalid
	}
	return nil
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

// validateAvatarImageDimensions проверяет размеры изображения по метаданным файла без декодирования всех пикселей.
func validateAvatarImageDimensions(content []byte) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return model.ErrInvalidAvatarMetadata
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return model.ErrInvalidAvatarMetadata
	}
	if cfg.Width > maxAvatarImageWidthPixels || cfg.Height > maxAvatarImageHeightPixels {
		return model.ErrInvalidAvatarMetadata
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxAvatarImageAreaPixels {
		return model.ErrInvalidAvatarMetadata
	}
	return nil
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
	if errors.Is(err, errAvatarFileNameInvalid) {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid file name", "")
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
	return url.JoinPath(apiPathPrefix, "avatars", avatarID.String())
}

// selectCurrentAvatar парсит X-User-ID и avatar_id из JSON-тела, затем передает их в сценарий выбора аватарки.
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

// listUserAvatars парсит user_id из пути и возвращает список аватарок пользователя.
func (h *AvatarHandler) listUserAvatars(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserIDPathParam(r)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid user_id", "")
		return
	}

	output, err := h.avatarUseCase.ListUserAvatars(r.Context(), usecase.ListUserAvatarsInput{
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			writeError(h.logger, w, r, http.StatusNotFound, "User not found", "")
			return
		}
		logError(h.logger, r, "failed to list user avatars", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	response, err := newListUserAvatarsResponse(output)
	if err != nil {
		logError(h.logger, r, "failed to build user avatars response", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	writeJSON(h.logger, w, r, http.StatusOK, response)
}

func parseUserIDPathParam(r *http.Request) (uuid.UUID, error) {
	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse user_id path param: %w", err)
	}
	if userID == uuid.Nil {
		return uuid.Nil, errors.New("invalid user_id path param")
	}
	return userID, nil
}

func newListUserAvatarsResponse(output usecase.ListUserAvatarsOutput) (dto.ListUserAvatarsResponse, error) {
	avatars := make([]dto.ListUserAvatarsItemResponse, 0, len(output.Avatars))
	for _, avatar := range output.Avatars {
		response, err := newListUserAvatarsItemResponse(avatar)
		if err != nil {
			return dto.ListUserAvatarsResponse{}, err
		}
		avatars = append(avatars, response)
	}

	return dto.ListUserAvatarsResponse{Avatars: avatars}, nil
}

func newListUserAvatarsItemResponse(
	avatar usecase.ListUserAvatarsItemOutput,
) (dto.ListUserAvatarsItemResponse, error) {
	avatarURL, err := avatarURLForID(avatar.ID)
	if err != nil {
		return dto.ListUserAvatarsItemResponse{}, err
	}

	return dto.ListUserAvatarsItemResponse{
		ID:        avatar.ID.String(),
		UserID:    avatar.UserID.String(),
		URL:       avatarURL,
		FileName:  avatar.FileName,
		MIMEType:  avatar.MIMEType,
		SizeBytes: avatar.SizeBytes,
		Width:     avatar.Width,
		Height:    avatar.Height,
		Status:    string(avatar.Status),
		IsCurrent: avatar.IsCurrent,
		CreatedAt: avatar.CreatedAt,
		UpdatedAt: avatar.UpdatedAt,
	}, nil
}

// getCurrentAvatarByEmail парсит email из query-параметра и возвращает текущую аватарку или PNG-заглушку.
func (h *AvatarHandler) getCurrentAvatarByEmail(w http.ResponseWriter, r *http.Request) {
	email, err := emailFromAvatarRequest(r)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid email", "")
		return
	}

	output, err := h.avatarUseCase.GetCurrentAvatarByEmail(r.Context(), usecase.GetCurrentAvatarByEmailInput{
		Email: email,
	})
	if err != nil {
		if errors.Is(err, model.ErrInvalidEmail) {
			writeError(h.logger, w, r, http.StatusBadRequest, "Invalid email", "")
			return
		}
		logError(h.logger, r, "failed to get current avatar by email", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	if output.UseDefaultAvatar {
		writeAvatarContent(h.logger, w, r, model.MIMEPNG, defaultAvatarPNG)
		return
	}
	writeAvatarContent(h.logger, w, r, output.MIMEType, output.Content)
}

func emailFromAvatarRequest(r *http.Request) (model.Email, error) {
	return model.NewEmail(r.URL.Query().Get("email"))
}

// getCurrentAvatarByUserID парсит user_id из пути и возвращает текущую аватарку или PNG-заглушку.
func (h *AvatarHandler) getCurrentAvatarByUserID(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserIDPathParam(r)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid user_id", "")
		return
	}

	output, err := h.avatarUseCase.GetCurrentAvatarByUserID(r.Context(), usecase.GetCurrentAvatarByUserIDInput{
		UserID: userID,
	})
	if err != nil {
		logError(h.logger, r, "failed to get current avatar by user id", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	if output.UseDefaultAvatar {
		writeAvatarContent(h.logger, w, r, model.MIMEPNG, defaultAvatarPNG)
		return
	}
	writeAvatarContent(h.logger, w, r, output.MIMEType, output.Content)
}

// writeAvatarContent выставляет одинаковые заголовки кеширования для пользовательской аватарки и PNG-заглушки.
func writeAvatarContent(logger *slog.Logger, w http.ResponseWriter, r *http.Request, mimeType string, content []byte) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", avatarCacheControl)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(content); err != nil {
		logger.ErrorContext(
			r.Context(),
			"failed to write HTTP response",
			slog.Any("error", err),
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.Int("status", http.StatusOK),
		)
	}
}

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
	avatarID, err := uuid.Parse(chi.URLParam(r, "avatar_id"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse avatar_id path param: %w", err)
	}
	if avatarID == uuid.Nil {
		return uuid.Nil, errors.New("invalid avatar_id path param")
	}
	return avatarID, nil
}

func avatarSizeFromRequest(r *http.Request) (usecase.AvatarSize, error) {
	switch r.URL.Query().Get("size") {
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
	switch r.URL.Query().Get("format") {
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

// getAvatarMetadata парсит avatar_id из пути и возвращает метаданные аватарки в JSON-ответе.
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
