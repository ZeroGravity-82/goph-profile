package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// MarkAvatarReady завершает успешную обработку аватарки.
func (uc *AvatarUseCase) MarkAvatarReady(
	ctx context.Context,
	in MarkAvatarReadyInput,
) (MarkAvatarReadyOutput, error) {
	if uc.userRepo == nil {
		return MarkAvatarReadyOutput{}, errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return MarkAvatarReadyOutput{}, errors.New("avatar repository is not provided")
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return MarkAvatarReadyOutput{}, fmt.Errorf("get avatar by id: %w", err)
	}

	now := time.Now().UTC()
	if err = avatar.MarkReady(
		in.Width,
		in.Height,
		in.ObjectKeyThumb100,
		in.ObjectKeyThumb300,
		now,
	); err != nil {
		return MarkAvatarReadyOutput{}, err
	}

	user, err := uc.userRepo.GetByID(ctx, avatar.UserID)
	if err != nil {
		return MarkAvatarReadyOutput{}, fmt.Errorf("get user by id: %w", err)
	}
	selectAsCurrent := user.CurrentAvatarID == nil
	if selectAsCurrent {
		if err = user.SelectCurrentAvatar(avatar, now); err != nil {
			return MarkAvatarReadyOutput{}, err
		}
	}

	if err = uc.avatarRepo.Update(ctx, avatar); err != nil {
		return MarkAvatarReadyOutput{}, fmt.Errorf("update ready avatar: %w", err)
	}
	if selectAsCurrent {
		if err = uc.userRepo.Update(ctx, user); err != nil {
			return MarkAvatarReadyOutput{}, fmt.Errorf("update current avatar: %w", err)
		}
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
