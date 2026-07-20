package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SelectCurrentAvatar выбирает готовую аватарку пользователя как текущую.
func (uc *AvatarUseCase) SelectCurrentAvatar(
	ctx context.Context,
	in SelectCurrentAvatarInput,
) error {
	if uc.userRepo == nil {
		return errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return errors.New("avatar repository is not provided")
	}

	user, err := uc.userRepo.GetByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("get user by id: %w", err)
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return fmt.Errorf("get avatar by id: %w", err)
	}

	alreadyCurrent := user.CurrentAvatarID != nil && *user.CurrentAvatarID == avatar.ID
	if err = user.SelectCurrentAvatar(avatar, time.Now().UTC()); err != nil {
		return err
	}
	if alreadyCurrent {
		return nil
	}

	if err = uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update current avatar: %w", err)
	}

	return nil
}
