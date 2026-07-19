package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// ResolveUserByEmailOutput описывает результат сценария ResolveUserByEmail.
type ResolveUserByEmailOutput struct {
	ID    uuid.UUID
	Email model.Email
}

// userRepository описывает операции с пользователем, которые нужны сценариям работы с пользователями.
type userRepository interface {
	GetByEmail(ctx context.Context, email model.Email) (model.User, error)
	Create(ctx context.Context, email model.Email) (model.User, error)
}

// UserUseCase реализует сценарии работы с пользователями.
type UserUseCase struct {
	userRepo userRepository
}

// NewUserUseCase создает UserUseCase.
func NewUserUseCase(userRepo userRepository) (*UserUseCase, error) {
	if userRepo == nil {
		return nil, errors.New("user repository is not provided")
	}
	return &UserUseCase{userRepo: userRepo}, nil
}

// ResolveUserByEmail возвращает пользователя по email или создает его.
//
// Конкурентное создание одного email решается через уникальность email в хранилище:
// если Create возвращает ErrEmailAlreadyTaken, сценарий повторно читает пользователя.
func (uc *UserUseCase) ResolveUserByEmail(
	ctx context.Context,
	email model.Email,
) (ResolveUserByEmailOutput, error) {
	if uc.userRepo == nil {
		return ResolveUserByEmailOutput{}, errors.New("user repository is not provided")
	}
	if err := email.Validate(); err != nil {
		return ResolveUserByEmailOutput{}, err
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return resolveUserByEmailOutput(user), nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return ResolveUserByEmailOutput{}, fmt.Errorf("get user by email: %w", err)
	}

	user, err = uc.userRepo.Create(ctx, email)
	if err == nil {
		return resolveUserByEmailOutput(user), nil
	}
	if !errors.Is(err, ErrEmailAlreadyTaken) {
		return ResolveUserByEmailOutput{}, fmt.Errorf("create user by email: %w", err)
	}

	user, err = uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return ResolveUserByEmailOutput{}, fmt.Errorf("get user after email conflict: %w", err)
	}

	return resolveUserByEmailOutput(user), nil
}

func resolveUserByEmailOutput(user model.User) ResolveUserByEmailOutput {
	return ResolveUserByEmailOutput{
		ID:    user.ID,
		Email: user.Email,
	}
}
