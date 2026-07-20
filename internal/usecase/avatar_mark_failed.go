package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// MarkAvatarFailed завершает обработку аватарки ошибкой.
func (uc *AvatarUseCase) MarkAvatarFailed(
	ctx context.Context,
	in MarkAvatarFailedInput,
) (MarkAvatarFailedOutput, error) {
	if uc.avatarRepo == nil {
		return MarkAvatarFailedOutput{}, errors.New("avatar repository is not provided")
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return MarkAvatarFailedOutput{}, fmt.Errorf("get avatar by id: %w", err)
	}

	if err = avatar.MarkFailed(time.Now().UTC()); err != nil {
		return MarkAvatarFailedOutput{}, err
	}

	if err = uc.avatarRepo.Update(ctx, avatar); err != nil {
		return MarkAvatarFailedOutput{}, fmt.Errorf("update failed avatar: %w", err)
	}

	return MarkAvatarFailedOutput{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		Status:    avatar.Status,
		UpdatedAt: avatar.UpdatedAt,
	}, nil
}
