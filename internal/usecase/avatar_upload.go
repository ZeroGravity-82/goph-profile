package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// UploadAvatar загружает аватарку.
func (uc *AvatarUseCase) UploadAvatar(ctx context.Context, in UploadAvatarInput) (UploadAvatarOutput, error) {
	if uc.userRepo == nil {
		return UploadAvatarOutput{}, errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return UploadAvatarOutput{}, errors.New("avatar repository is not provided")
	}
	if uc.fileStorage == nil {
		return UploadAvatarOutput{}, errors.New("file storage is not provided")
	}
	if uc.publisher == nil {
		return UploadAvatarOutput{}, errors.New("avatar message publisher is not provided")
	}

	uuidV7, err := uuid.NewV7()
	if err != nil {
		return UploadAvatarOutput{}, fmt.Errorf("failed to create avatar id: %w", err)
	}
	objectKeyOriginal := uc.fileStorage.ObjectKey(in.UserID, uuidV7)
	avatar, err := model.NewProcessingAvatar(
		uuidV7,
		in.UserID,
		in.FileName,
		in.MIMEType,
		int64(len(in.Content)),
		objectKeyOriginal,
		time.Now().UTC(),
	)
	if err != nil {
		return UploadAvatarOutput{}, err
	}

	if _, err = uc.userRepo.GetByID(ctx, in.UserID); err != nil {
		return UploadAvatarOutput{}, fmt.Errorf("get user by id: %w", err)
	}

	if err = uc.fileStorage.Put(ctx, objectKeyOriginal, in.Content); err != nil {
		return UploadAvatarOutput{}, fmt.Errorf("put original avatar: %w", err)
	}

	if err = uc.avatarRepo.Create(ctx, avatar); err != nil {
		deleteErr := uc.fileStorage.Delete(ctx, objectKeyOriginal)
		if deleteErr != nil {
			return UploadAvatarOutput{}, fmt.Errorf(
				"failed to delete uploaded original after create avatar error: %w; delete error: %w",
				err,
				deleteErr,
			)
		}
		return UploadAvatarOutput{}, fmt.Errorf("create avatar: %w", err)
	}

	message := AvatarProcessingMessage{
		AvatarID:          avatar.ID,
		UserID:            avatar.UserID,
		ObjectKeyOriginal: avatar.ObjectKeyOriginal,
	}
	if err = uc.publisher.PublishAvatarProcessing(ctx, message); err != nil {
		deleteAvatarErr := uc.avatarRepo.Delete(ctx, avatar.ID)
		deleteObjectErr := uc.fileStorage.Delete(ctx, objectKeyOriginal)

		if deleteAvatarErr != nil && deleteObjectErr != nil {
			return UploadAvatarOutput{}, fmt.Errorf(
				"failed to delete avatar and uploaded original after publish avatar processing message error: %w; "+
					"delete avatar error: %w; delete error: %w",
				err,
				deleteAvatarErr,
				deleteObjectErr,
			)
		}
		if deleteAvatarErr != nil {
			return UploadAvatarOutput{}, fmt.Errorf(
				"failed to delete avatar after publish avatar processing message error: %w; delete avatar error: %w",
				err,
				deleteAvatarErr,
			)
		}
		if deleteObjectErr != nil {
			return UploadAvatarOutput{}, fmt.Errorf(
				"failed to delete uploaded original after publish avatar processing message error: %w; delete error: %w",
				err,
				deleteObjectErr,
			)
		}

		return UploadAvatarOutput{}, fmt.Errorf("publish avatar processing message: %w", err)
	}

	return UploadAvatarOutput{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		FileName:  avatar.FileName,
		MIMEType:  avatar.MIMEType,
		SizeBytes: avatar.SizeBytes,
		Status:    avatar.Status,
		CreatedAt: avatar.CreatedAt,
		UpdatedAt: avatar.UpdatedAt,
	}, nil
}
