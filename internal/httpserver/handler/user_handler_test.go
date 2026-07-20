package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/httpserver/dto"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

// TestUserHandler_resolveUserByEmail проверяет успешное определение пользователя по email.
func TestUserHandler_resolveUserByEmail(t *testing.T) {
	// Arrange
	userUseCase := &userUseCaseFake{
		resolveOutput: usecase.ResolveUserByEmailOutput{
			ID:    testUserID,
			Email: model.Email("user@example.com"),
		},
	}
	handler := NewUserHandler(userUseCase, discardLogger())
	request := newResolveUserRequest(t, "  User@Example.COM  ")
	response := httptest.NewRecorder()

	// Act
	handler.resolveUserByEmail(response, request)

	// Assert
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.Equal(t, []model.Email{"user@example.com"}, userUseCase.resolveInputs)

	var body dto.ResolveUserResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, testUserID.String(), body.UserID)
	assert.Equal(t, "user@example.com", body.Email)
}

// TestUserHandler_resolveUserByEmail_RejectsInvalidJSON проверяет ошибку невалидного JSON-тела запроса.
func TestUserHandler_resolveUserByEmail_RejectsInvalidJSON(t *testing.T) {
	// Arrange
	userUseCase := &userUseCaseFake{}
	handler := NewUserHandler(userUseCase, discardLogger())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users/resolve", bytes.NewBufferString("{"))
	response := httptest.NewRecorder()

	// Act
	handler.resolveUserByEmail(response, request)

	// Assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, userUseCase.resolveInputs)
	assertErrorResponse(t, response, "Invalid request body")
}

// TestUserHandler_resolveUserByEmail_RejectsInvalidEmail проверяет ошибку невалидного email.
func TestUserHandler_resolveUserByEmail_RejectsInvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{name: "empty", email: " "},
		{name: "without at", email: "user.example.com"},
		{name: "empty local", email: "@example.com"},
		{name: "empty domain", email: "user@"},
		{name: "spaces", email: "user name@example.com"},
		{name: "too long", email: strings.Repeat("a", model.MaxEmailSizeBytes-10) + "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			userUseCase := &userUseCaseFake{}
			handler := NewUserHandler(userUseCase, discardLogger())
			request := newResolveUserRequest(t, tt.email)
			response := httptest.NewRecorder()

			// Act
			handler.resolveUserByEmail(response, request)

			// Assert
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Empty(t, userUseCase.resolveInputs)
			assertErrorResponse(t, response, "Invalid email")
		})
	}
}

// TestUserHandler_resolveUserByEmail_ReturnsInternalServerError проверяет внутреннюю ошибку определения пользователя.
func TestUserHandler_resolveUserByEmail_ReturnsInternalServerError(t *testing.T) {
	// Arrange
	userUseCase := &userUseCaseFake{resolveErr: errors.New("database error")}
	handler := NewUserHandler(userUseCase, discardLogger())
	request := newResolveUserRequest(t, "user@example.com")
	response := httptest.NewRecorder()

	// Act
	handler.resolveUserByEmail(response, request)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Equal(t, []model.Email{"user@example.com"}, userUseCase.resolveInputs)
	assertErrorResponse(t, response, "Internal server error")
}
