package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// GetCurrentAvatarByEmail возвращает текущую готовую аватарку пользователя или признак выдачи заглушки.
func (uc *AvatarUseCase) GetCurrentAvatarByEmail(
	ctx context.Context,
	in GetCurrentAvatarByEmailInput,
) (GetCurrentAvatarByEmailOutput, error) {
	if uc.userRepo == nil {
		return GetCurrentAvatarByEmailOutput{}, errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return GetCurrentAvatarByEmailOutput{}, errors.New("avatar repository is not provided")
	}
	if uc.fileStorage == nil {
		return GetCurrentAvatarByEmailOutput{}, errors.New("file storage is not provided")
	}
	if err := in.Email.Validate(); err != nil {
		return GetCurrentAvatarByEmailOutput{}, err
	}

	user, err := uc.userRepo.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
		}
		return GetCurrentAvatarByEmailOutput{}, fmt.Errorf("get user by email: %w", err)
	}
	if user.CurrentAvatarID == nil {
		return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, *user.CurrentAvatarID)
	if err != nil {
		if errors.Is(err, ErrAvatarNotFound) {
			return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
		}
		return GetCurrentAvatarByEmailOutput{}, fmt.Errorf("get current avatar by id: %w", err)
	}
	if avatar.UserID != user.ID {
		return GetCurrentAvatarByEmailOutput{}, model.ErrAvatarForbidden
	}
	if err = avatar.CanBeCurrent(); err != nil {
		return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
	}

	content, err := uc.fileStorage.Get(ctx, avatar.ObjectKeyOriginal)
	if err != nil {
		return GetCurrentAvatarByEmailOutput{}, fmt.Errorf("get original avatar object: %w", err)
	}

	return GetCurrentAvatarByEmailOutput{
		Content:  content,
		MIMEType: avatar.MIMEType,
	}, nil
}
