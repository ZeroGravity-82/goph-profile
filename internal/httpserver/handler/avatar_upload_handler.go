package handler

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

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
