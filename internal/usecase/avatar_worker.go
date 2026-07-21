package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// MarkAvatarReadyInput описывает входные данные сценария завершения обработки аватарки.
type MarkAvatarReadyInput struct {
	AvatarID          uuid.UUID
	Width             int
	Height            int
	ObjectKeyThumb100 string
	ObjectKeyThumb300 string
}

// MarkAvatarReadyOutput описывает результат сценария завершения обработки аватарки.
type MarkAvatarReadyOutput struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Width             int
	Height            int
	ObjectKeyThumb100 string
	ObjectKeyThumb300 string
	Status            model.AvatarStatus
	UpdatedAt         time.Time
}

// MarkAvatarFailedInput описывает входные данные сценария ошибки обработки аватарки.
type MarkAvatarFailedInput struct {
	AvatarID uuid.UUID
}

// MarkAvatarFailedOutput описывает результат сценария ошибки обработки аватарки.
type MarkAvatarFailedOutput struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Status    model.AvatarStatus
	UpdatedAt time.Time
}

// MarkAvatarDeletedInput описывает входные данные сценария завершения удаления файлов аватарки.
type MarkAvatarDeletedInput struct {
	AvatarID uuid.UUID
}

// AvatarWorkerUseCase реализует сценарии обработки аватарок воркером.
type AvatarWorkerUseCase struct {
	userRepo   avatarUserRepository
	avatarRepo avatarRepository
	transactor transactor
}

// NewAvatarWorkerUseCase создает AvatarWorkerUseCase.
func NewAvatarWorkerUseCase(
	userRepo avatarUserRepository,
	avatarRepo avatarRepository,
	transactor transactor,
) (*AvatarWorkerUseCase, error) {
	if userRepo == nil {
		return nil, errors.New("user repository is not provided")
	}
	if avatarRepo == nil {
		return nil, errors.New("avatar repository is not provided")
	}
	if transactor == nil {
		return nil, errors.New("transactor is not provided")
	}

	return &AvatarWorkerUseCase{
		userRepo:   userRepo,
		avatarRepo: avatarRepo,
		transactor: transactor,
	}, nil
}

// MarkAvatarReady завершает успешную обработку аватарки.
func (uc *AvatarWorkerUseCase) MarkAvatarReady(
	ctx context.Context,
	in MarkAvatarReadyInput,
) (MarkAvatarReadyOutput, error) {
	var avatar model.Avatar
	if err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		avatar, err = uc.avatarRepo.GetByID(txCtx, in.AvatarID)
		if err != nil {
			return fmt.Errorf("failed to get avatar by id: %w", err)
		}

		now := time.Now().UTC()
		if err = avatar.MarkReady(
			in.Width,
			in.Height,
			in.ObjectKeyThumb100,
			in.ObjectKeyThumb300,
			now,
		); err != nil {
			return err
		}

		user, err := uc.userRepo.GetByID(txCtx, avatar.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user by id: %w", err)
		}
		selectAsCurrent := user.CurrentAvatarID == nil
		if selectAsCurrent {
			if err = user.SelectCurrentAvatar(avatar, now); err != nil {
				return err
			}
		}

		if err = uc.avatarRepo.Update(txCtx, avatar); err != nil {
			return fmt.Errorf("failed to update ready avatar: %w", err)
		}
		if selectAsCurrent {
			if err = uc.userRepo.Update(txCtx, user); err != nil {
				return fmt.Errorf("failed to update current avatar: %w", err)
			}
		}
		return nil
	}); err != nil {
		return MarkAvatarReadyOutput{}, err
	}

	return MarkAvatarReadyOutput{
		ID:                avatar.ID,
		UserID:            avatar.UserID,
		Width:             *avatar.Width,
		Height:            *avatar.Height,
		ObjectKeyThumb100: *avatar.ObjectKeyThumb100,
		ObjectKeyThumb300: *avatar.ObjectKeyThumb300,
		Status:            avatar.Status,
		UpdatedAt:         avatar.UpdatedAt,
	}, nil
}

// MarkAvatarFailed завершает обработку аватарки ошибкой.
func (uc *AvatarWorkerUseCase) MarkAvatarFailed(
	ctx context.Context,
	in MarkAvatarFailedInput,
) (MarkAvatarFailedOutput, error) {
	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return MarkAvatarFailedOutput{}, fmt.Errorf("failed to get avatar by id: %w", err)
	}

	if err = avatar.MarkFailed(time.Now().UTC()); err != nil {
		return MarkAvatarFailedOutput{}, err
	}

	if err = uc.avatarRepo.Update(ctx, avatar); err != nil {
		return MarkAvatarFailedOutput{}, fmt.Errorf("failed to update failed avatar: %w", err)
	}

	return MarkAvatarFailedOutput{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		Status:    avatar.Status,
		UpdatedAt: avatar.UpdatedAt,
	}, nil
}

// MarkAvatarDeleted завершает удаление файлов аватарки.
func (uc *AvatarWorkerUseCase) MarkAvatarDeleted(ctx context.Context, in MarkAvatarDeletedInput) error {
	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return fmt.Errorf("failed to get avatar by id: %w", err)
	}

	if err = avatar.MarkDeleted(time.Now().UTC()); err != nil {
		return err
	}

	if err = uc.avatarRepo.Update(ctx, avatar); err != nil {
		return fmt.Errorf("failed to update deleted avatar: %w", err)
	}

	return nil
}
