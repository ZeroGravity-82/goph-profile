package httpserver

import (
	"context"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

type avatarUseCaseFake struct{}

func (uc *avatarUseCaseFake) UploadAvatar(
	_ context.Context,
	_ usecase.UploadAvatarInput,
) (usecase.UploadAvatarOutput, error) {
	return usecase.UploadAvatarOutput{}, nil
}

func (uc *avatarUseCaseFake) SelectCurrentAvatar(_ context.Context, _ usecase.SelectCurrentAvatarInput) error {
	return nil
}

func (uc *avatarUseCaseFake) DeleteCurrentAvatar(_ context.Context, _ usecase.DeleteCurrentAvatarInput) error {
	return nil
}

func (uc *avatarUseCaseFake) DeleteAvatar(_ context.Context, _ usecase.DeleteAvatarInput) error {
	return nil
}

func (uc *avatarUseCaseFake) ListUserAvatars(
	_ context.Context,
	_ usecase.ListUserAvatarsInput,
) (usecase.ListUserAvatarsOutput, error) {
	return usecase.ListUserAvatarsOutput{}, nil
}

func (uc *avatarUseCaseFake) GetCurrentAvatarByEmail(
	_ context.Context,
	_ usecase.GetCurrentAvatarByEmailInput,
) (usecase.GetCurrentAvatarByEmailOutput, error) {
	return usecase.GetCurrentAvatarByEmailOutput{}, nil
}

func (uc *avatarUseCaseFake) GetCurrentAvatarByUserID(
	_ context.Context,
	_ usecase.GetCurrentAvatarByUserIDInput,
) (usecase.GetCurrentAvatarByUserIDOutput, error) {
	return usecase.GetCurrentAvatarByUserIDOutput{}, nil
}

func (uc *avatarUseCaseFake) GetAvatar(
	_ context.Context,
	_ usecase.GetAvatarInput,
) (usecase.GetAvatarOutput, error) {
	return usecase.GetAvatarOutput{}, nil
}

func (uc *avatarUseCaseFake) GetAvatarMetadata(
	_ context.Context,
	_ usecase.GetAvatarMetadataInput,
) (usecase.GetAvatarMetadataOutput, error) {
	return usecase.GetAvatarMetadataOutput{}, nil
}

type userUseCaseFake struct{}

func (uc *userUseCaseFake) ResolveUserByEmail(
	_ context.Context,
	_ model.Email,
) (usecase.ResolveUserByEmailOutput, error) {
	return usecase.ResolveUserByEmailOutput{}, nil
}
