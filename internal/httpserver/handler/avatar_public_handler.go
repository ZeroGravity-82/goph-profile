package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// getPublicAvatarByEmail парсит email из query-параметра и возвращает текущую аватарку или PNG-заглушку.
func (h *AvatarHandler) getPublicAvatarByEmail(w http.ResponseWriter, r *http.Request) {
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
		logError(h.logger, r, "failed to get public avatar by email", err)
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

// writeAvatarContent выставляет одинаковые заголовки кеширования для пользовательской аватарки и PNG-заглушки.
func writeAvatarContent(logger *slog.Logger, w http.ResponseWriter, r *http.Request, mimeType string, content []byte) {
	if logger == nil {
		logger = logging.NopLogger()
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", publicAvatarCacheControl)
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
