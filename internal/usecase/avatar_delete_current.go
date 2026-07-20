package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// DeleteCurrentAvatar удаляет текущую аватарку пользователя.
func (uc *AvatarUseCase) DeleteCurrentAvatar(ctx context.Context, in DeleteCurrentAvatarInput) error {
	if uc.userRepo == nil {
		return errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return errors.New("avatar repository is not provided")
	}
	if uc.publisher == nil {
		return errors.New("avatar message publisher is not provided")
	}

	user, err := uc.userRepo.GetByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("get user by id: %w", err)
	}
	if user.CurrentAvatarID == nil {
		return nil
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, *user.CurrentAvatarID)
	if err != nil {
		return fmt.Errorf("get current avatar by id: %w", err)
	}
	if avatar.UserID != user.ID {
		return model.ErrAvatarForbidden
	}

	now := time.Now().UTC()
	if err = avatar.MarkDeleting(now); err != nil {
		return err
	}
	user.ClearCurrentAvatar(avatar.ID, now)

	if err = uc.avatarRepo.Update(ctx, avatar); err != nil {
		return fmt.Errorf("update deleting avatar: %w", err)
	}
	if err = uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("clear current avatar: %w", err)
	}

	message := AvatarDeletionMessage{
		AvatarID:   avatar.ID,
		ObjectKeys: avatarObjectKeys(avatar),
	}
	if err = uc.publisher.PublishAvatarDeletion(ctx, message); err != nil {
		return fmt.Errorf("publish avatar deletion message: %w", err)
	}

	return nil
}

func avatarObjectKeys(avatar model.Avatar) []string {
	objectKeys := []string{avatar.ObjectKeyOriginal}
	if avatar.ObjectKeyThumb100 != nil {
		objectKeys = append(objectKeys, *avatar.ObjectKeyThumb100)
	}
	if avatar.ObjectKeyThumb300 != nil {
		objectKeys = append(objectKeys, *avatar.ObjectKeyThumb300)
	}
	return objectKeys
}
