package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

const maxEmailSizeBytes = 255

// userUseCase описывает сценарии работы с пользователями: определение пользователя по email.
type userUseCase interface {
	ResolveUserByEmail(ctx context.Context, email model.Email) (usecase.ResolveUserByEmailOutput, error)
}

// UserHandler обрабатывает HTTP-запросы для пользователей.
type UserHandler struct {
	userUseCase userUseCase
	logger      *slog.Logger
}

// NewUserHandler создает UserHandler.
func NewUserHandler(
	userUseCase userUseCase,
	logger *slog.Logger,
) (*UserHandler, error) {
	if userUseCase == nil {
		return nil, errors.New("user usecase is not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &UserHandler{
		userUseCase: userUseCase,
		logger:      logger,
	}, nil
}

// resolveUserByEmail парсит JSON-запрос и передает email в сценарий определения пользователя.
func (h *UserHandler) resolveUserByEmail(w http.ResponseWriter, r *http.Request) {
	request, err := resolveUserRequestFromRequest(r)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid request body", "")
		return
	}
	if len(request.Email) > maxEmailSizeBytes {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid email", "")
		return
	}
	email, err := model.NewEmail(request.Email)
	if err != nil {
		writeError(h.logger, w, r, http.StatusBadRequest, "Invalid email", "")
		return
	}

	output, err := h.userUseCase.ResolveUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, model.ErrInvalidEmail) {
			writeError(h.logger, w, r, http.StatusBadRequest, "Invalid email", "")
			return
		}
		logError(h.logger, r, "failed to resolve user by email", err)
		writeError(h.logger, w, r, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	writeJSON(h.logger, w, r, http.StatusOK, dto.ResolveUserResponse{
		UserID: output.ID.String(),
		Email:  string(output.Email),
	})
}

func resolveUserRequestFromRequest(r *http.Request) (dto.ResolveUserRequest, error) {
	var request dto.ResolveUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return dto.ResolveUserRequest{}, err
	}
	return request, nil
}
