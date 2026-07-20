package handler

import (
	"context"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

type avatarUseCaseFake struct {
	uploadOutput         usecase.UploadAvatarOutput
	uploadErr            error
	uploadInputs         []usecase.UploadAvatarInput
	currentByEmailOutput usecase.GetCurrentAvatarByEmailOutput
	currentByEmailErr    error
	currentByEmailInputs []usecase.GetCurrentAvatarByEmailInput
	metadataOutput       usecase.GetAvatarMetadataOutput
	metadataErr          error
	metadataInputs       []usecase.GetAvatarMetadataInput
}

func (uc *avatarUseCaseFake) UploadAvatar(
	_ context.Context,
	input usecase.UploadAvatarInput,
) (usecase.UploadAvatarOutput, error) {
	uc.uploadInputs = append(uc.uploadInputs, input)
	return uc.uploadOutput, uc.uploadErr
}

func (uc *avatarUseCaseFake) GetCurrentAvatarByEmail(
	_ context.Context,
	input usecase.GetCurrentAvatarByEmailInput,
) (usecase.GetCurrentAvatarByEmailOutput, error) {
	uc.currentByEmailInputs = append(uc.currentByEmailInputs, input)
	return uc.currentByEmailOutput, uc.currentByEmailErr
}

func (uc *avatarUseCaseFake) GetAvatarMetadata(
	_ context.Context,
	input usecase.GetAvatarMetadataInput,
) (usecase.GetAvatarMetadataOutput, error) {
	uc.metadataInputs = append(uc.metadataInputs, input)
	return uc.metadataOutput, uc.metadataErr
}

type userUseCaseFake struct {
	resolveOutput usecase.ResolveUserByEmailOutput
	resolveErr    error
	resolveInputs []model.Email
}

func (uc *userUseCaseFake) ResolveUserByEmail(
	_ context.Context,
	email model.Email,
) (usecase.ResolveUserByEmailOutput, error) {
	uc.resolveInputs = append(uc.resolveInputs, email)
	return uc.resolveOutput, uc.resolveErr
}
