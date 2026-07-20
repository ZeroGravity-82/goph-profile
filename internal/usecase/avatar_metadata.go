package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// GetAvatarMetadata возвращает метаданные аватарки.
func (uc *AvatarUseCase) GetAvatarMetadata(
	ctx context.Context,
	in GetAvatarMetadataInput,
) (GetAvatarMetadataOutput, error) {
	if uc.avatarRepo == nil {
		return GetAvatarMetadataOutput{}, errors.New("avatar repository is not provided")
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return GetAvatarMetadataOutput{}, fmt.Errorf("get avatar by id: %w", err)
	}

	return getAvatarMetadataOutput(avatar), nil
}

func getAvatarMetadataOutput(avatar model.Avatar) GetAvatarMetadataOutput {
	return GetAvatarMetadataOutput{
		ID:                avatar.ID,
		UserID:            avatar.UserID,
		FileName:          avatar.FileName,
		MIMEType:          avatar.MIMEType,
		SizeBytes:         avatar.SizeBytes,
		Width:             avatar.Width,
		Height:            avatar.Height,
		ObjectKeyOriginal: avatar.ObjectKeyOriginal,
		ObjectKeyThumb100: avatar.ObjectKeyThumb100,
		ObjectKeyThumb300: avatar.ObjectKeyThumb300,
		Status:            avatar.Status,
		CreatedAt:         avatar.CreatedAt,
		UpdatedAt:         avatar.UpdatedAt,
		DeletedAt:         avatar.DeletedAt,
	}
}
