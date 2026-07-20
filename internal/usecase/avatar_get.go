package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// GetAvatar возвращает файл готовой неудаленной аватарки.
func (uc *AvatarUseCase) GetAvatar(ctx context.Context, in GetAvatarInput) (GetAvatarOutput, error) {
	if uc.avatarRepo == nil {
		return GetAvatarOutput{}, errors.New("avatar repository is not provided")
	}
	if uc.fileStorage == nil {
		return GetAvatarOutput{}, errors.New("file storage is not provided")
	}
	if in.AvatarID == uuid.Nil {
		return GetAvatarOutput{}, model.ErrInvalidID
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return GetAvatarOutput{}, fmt.Errorf("get avatar by id: %w", err)
	}
	if err = avatar.CanBeCurrent(); err != nil {
		return GetAvatarOutput{}, ErrAvatarNotFound
	}

	objectKey, mimeType, err := avatarObjectForSize(avatar, in.Size)
	if err != nil {
		return GetAvatarOutput{}, err
	}
	if in.MIMEType != "" && in.MIMEType != mimeType {
		return GetAvatarOutput{}, model.ErrInvalidAvatarMetadata
	}

	content, err := uc.fileStorage.Get(ctx, objectKey)
	if err != nil {
		return GetAvatarOutput{}, fmt.Errorf("get avatar object: %w", err)
	}

	return GetAvatarOutput{
		Content:  content,
		MIMEType: mimeType,
	}, nil
}

// avatarObjectForSize выбирает ключ объекта и фактический MIME-тип файла: оригинал сохраняет исходный MIME-тип,
// миниатюры отдаются как PNG.
func avatarObjectForSize(avatar model.Avatar, size AvatarSize) (string, string, error) {
	switch size {
	case AvatarSizeOriginal:
		return avatar.ObjectKeyOriginal, avatar.MIMEType, nil
	case AvatarSize100:
		if avatar.ObjectKeyThumb100 == nil {
			return "", "", ErrAvatarNotFound
		}
		return *avatar.ObjectKeyThumb100, model.MIMEPNG, nil
	case AvatarSize300:
		if avatar.ObjectKeyThumb300 == nil {
			return "", "", ErrAvatarNotFound
		}
		return *avatar.ObjectKeyThumb300, model.MIMEPNG, nil
	default:
		return "", "", model.ErrInvalidAvatarMetadata
	}
}
