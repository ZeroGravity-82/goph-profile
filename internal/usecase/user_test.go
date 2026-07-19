package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

var testUserID = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a001")

type userRepositoryFake struct {
	getUser      model.User
	getUsers     []model.User
	getErr       error
	getErrs      []error
	createUser   model.User
	createErr    error
	getEmails    []model.Email
	createEmails []model.Email
}

func (r *userRepositoryFake) GetByEmail(_ context.Context, email model.Email) (model.User, error) {
	r.getEmails = append(r.getEmails, email)
	if len(r.getUsers) > 0 {
		user := r.getUsers[0]
		r.getUsers = r.getUsers[1:]
		err := r.nextGetErr()
		return user, err
	}
	return r.getUser, r.getErr
}

func (r *userRepositoryFake) Create(_ context.Context, email model.Email) (model.User, error) {
	r.createEmails = append(r.createEmails, email)
	return r.createUser, r.createErr
}

func (r *userRepositoryFake) nextGetErr() error {
	if len(r.getErrs) == 0 {
		return r.getErr
	}
	err := r.getErrs[0]
	r.getErrs = r.getErrs[1:]
	return err
}

// TestNewUserUseCase_RejectsNilRepository проверяет обязательность репозитория.
func TestNewUserUseCase_RejectsNilRepository(t *testing.T) {
	// Act
	service, err := NewUserUseCase(nil)

	// Assert
	require.EqualError(t, err, "user repository is not provided")
	assert.Nil(t, service)
}

// TestUserUseCase_ResolveUserByEmail проверяет нормализацию email.
func TestUserUseCase_ResolveUserByEmail(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	user, err := model.NewUser(testUserID, "user@example.com", now)
	require.NoError(t, err)
	repository := &userRepositoryFake{getUser: user}
	service, err := NewUserUseCase(repository)
	require.NoError(t, err)

	// Act
	result, err := service.ResolveUserByEmail(ctx, "  User@Example.COM  ")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testUserID, result.ID)
	assert.Equal(t, model.Email("user@example.com"), result.Email)
	assert.Equal(t, []model.Email{"user@example.com"}, repository.getEmails)
	assert.Empty(t, repository.createEmails)
}

// TestUserUseCase_ResolveUserByEmail_CreatesUser проверяет создание пользователя.
func TestUserUseCase_ResolveUserByEmail_CreatesUser(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	user, err := model.NewUser(testUserID, "user@example.com", now)
	require.NoError(t, err)
	repository := &userRepositoryFake{
		getErr:     ErrUserNotFound,
		createUser: user,
	}
	service, err := NewUserUseCase(repository)
	require.NoError(t, err)

	// Act
	result, err := service.ResolveUserByEmail(ctx, "user@example.com")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testUserID, result.ID)
	assert.Equal(t, model.Email("user@example.com"), result.Email)
	assert.Equal(t, []model.Email{"user@example.com"}, repository.getEmails)
	assert.Equal(t, []model.Email{"user@example.com"}, repository.createEmails)
}

// TestUserUseCase_ResolveUserByEmail_ReadsAfterConflict проверяет чтение после конфликта.
func TestUserUseCase_ResolveUserByEmail_ReadsAfterConflict(t *testing.T) {
	// Arrange
	ctx := context.Background()
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	user, err := model.NewUser(testUserID, "user@example.com", now)
	require.NoError(t, err)
	repository := &userRepositoryFake{
		getUsers:   []model.User{{}, user},
		getErrs:    []error{ErrUserNotFound, nil},
		createErr:  ErrEmailAlreadyTaken,
		createUser: user,
	}
	service, err := NewUserUseCase(repository)
	require.NoError(t, err)

	// Act
	result, err := service.ResolveUserByEmail(ctx, "user@example.com")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testUserID, result.ID)
	assert.Equal(t, model.Email("user@example.com"), result.Email)
	assert.Equal(
		t,
		[]model.Email{"user@example.com", "user@example.com"},
		repository.getEmails,
	)
	assert.Equal(t, []model.Email{"user@example.com"}, repository.createEmails)
}

// TestUserUseCase_ResolveUserByEmail_RejectsInvalidEmail проверяет невалидный email.
func TestUserUseCase_ResolveUserByEmail_RejectsInvalidEmail(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty", raw: " "},
		{name: "without at", raw: "user.example.com"},
		{name: "empty local", raw: "@example.com"},
		{name: "empty domain", raw: "user@"},
		{name: "spaces", raw: "user name@example.com"},
		{name: "too long", raw: strings.Repeat("a", MaxEmailLength-10) + "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			repository := &userRepositoryFake{}
			service, err := NewUserUseCase(repository)
			require.NoError(t, err)

			// Act
			result, err := service.ResolveUserByEmail(ctx, tt.raw)

			// Assert
			require.ErrorIs(t, err, ErrInvalidEmail)
			assert.Zero(t, result)
			assert.Empty(t, repository.getEmails)
			assert.Empty(t, repository.createEmails)
		})
	}
}

// TestUserUseCase_ResolveUserByEmail_ReturnsGetError проверяет ошибку чтения.
func TestUserUseCase_ResolveUserByEmail_ReturnsGetError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repositoryErr := errors.New("repository error")
	repository := &userRepositoryFake{getErr: repositoryErr}
	service, err := NewUserUseCase(repository)
	require.NoError(t, err)

	// Act
	result, err := service.ResolveUserByEmail(ctx, "user@example.com")

	// Assert
	require.ErrorIs(t, err, repositoryErr)
	assert.Zero(t, result)
	assert.Equal(t, []model.Email{"user@example.com"}, repository.getEmails)
	assert.Empty(t, repository.createEmails)
}

// TestUserUseCase_ResolveUserByEmail_ReturnsCreateError проверяет ошибку создания.
func TestUserUseCase_ResolveUserByEmail_ReturnsCreateError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repositoryErr := errors.New("repository error")
	repository := &userRepositoryFake{
		getErr:    ErrUserNotFound,
		createErr: repositoryErr,
	}
	service, err := NewUserUseCase(repository)
	require.NoError(t, err)

	// Act
	result, err := service.ResolveUserByEmail(ctx, "user@example.com")

	// Assert
	require.ErrorIs(t, err, repositoryErr)
	assert.Zero(t, result)
	assert.Equal(t, []model.Email{"user@example.com"}, repository.getEmails)
	assert.Equal(t, []model.Email{"user@example.com"}, repository.createEmails)
}
