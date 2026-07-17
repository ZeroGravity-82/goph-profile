package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

// UploadAvatarInput описывает входные данные сценария загрузки аватарки.
type UploadAvatarInput struct {
	UserID   uuid.UUID
	FileName string
	MIMEType string
	Content  []byte
}

// UploadAvatarOutput описывает результат сценария загрузки аватарки.
type UploadAvatarOutput struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FileName  string
	MIMEType  string
	SizeBytes int64
	Status    model.AvatarStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// avatarUserRepository описывает операции с пользователем, которые нужны сценариям работы с аватарками.
type avatarUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.User, error)
}

// avatarRepository описывает нужные операции с аватарками.
type avatarRepository interface {
	Create(ctx context.Context, avatar model.Avatar) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// fileStorage описывает операции с хранилищем файлов.
type fileStorage interface {
	ObjectKey(userID uuid.UUID, avatarID uuid.UUID) string
	Put(ctx context.Context, objectKey string, content []byte) error
	Delete(ctx context.Context, objectKey string) error
}

// AvatarProcessingMessage описывает сообщение обработки.
type AvatarProcessingMessage struct {
	AvatarID          uuid.UUID
	UserID            uuid.UUID
	ObjectKeyOriginal string
}

// avatarMessagePublisher описывает публикацию сообщений обработки.
type avatarMessagePublisher interface {
	PublishAvatarProcessing(ctx context.Context, message AvatarProcessingMessage) error
}

// AvatarUseCase реализует сценарии работы с аватарками.
type AvatarUseCase struct {
	userRepo    avatarUserRepository
	avatarRepo  avatarRepository
	fileStorage fileStorage
	publisher   avatarMessagePublisher
}

// NewAvatarUseCase создает AvatarUseCase.
func NewAvatarUseCase(
	userRepo avatarUserRepository,
	avatarRepo avatarRepository,
	fileStorage fileStorage,
	messagePublisher avatarMessagePublisher,
) (*AvatarUseCase, error) {
	if userRepo == nil {
		return nil, errors.New("user repository is not provided")
	}
	if avatarRepo == nil {
		return nil, errors.New("avatar repository is not provided")
	}
	if fileStorage == nil {
		return nil, errors.New("file storage is not provided")
	}
	if messagePublisher == nil {
		return nil, errors.New("avatar message publisher is not provided")
	}

	return &AvatarUseCase{
		userRepo:    userRepo,
		avatarRepo:  avatarRepo,
		fileStorage: fileStorage,
		publisher:   messagePublisher,
	}, nil
}

// UploadAvatar загружает аватарку.
func (uc *AvatarUseCase) UploadAvatar(ctx context.Context, in UploadAvatarInput) (UploadAvatarOutput, error) {
	if uc == nil || uc.userRepo == nil {
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
		deleteErr := wrapCleanupError(
			"delete uploaded original after create failure",
			uc.fileStorage.Delete(ctx, objectKeyOriginal),
		)
		return UploadAvatarOutput{}, joinUploadError(fmt.Errorf("create avatar: %w", err), deleteErr)
	}

	message := AvatarProcessingMessage{
		AvatarID:          avatar.ID,
		UserID:            avatar.UserID,
		ObjectKeyOriginal: avatar.ObjectKeyOriginal,
	}
	if err = uc.publisher.PublishAvatarProcessing(ctx, message); err != nil {
		deleteAvatarErr := wrapCleanupError("delete avatar after publish failure", uc.avatarRepo.Delete(ctx, avatar.ID))
		deleteObjectErr := wrapCleanupError(
			"delete uploaded original after publish failure",
			uc.fileStorage.Delete(ctx, objectKeyOriginal),
		)
		publishErr := fmt.Errorf("publish avatar processing message: %w", err)

		return UploadAvatarOutput{}, joinUploadError(publishErr, deleteAvatarErr, deleteObjectErr)
	}

	return uploadAvatarOutput(avatar), nil
}

func uploadAvatarOutput(avatar model.Avatar) UploadAvatarOutput {
	return UploadAvatarOutput{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		FileName:  avatar.FileName,
		MIMEType:  avatar.MIMEType,
		SizeBytes: avatar.SizeBytes,
		Status:    avatar.Status,
		CreatedAt: avatar.CreatedAt,
		UpdatedAt: avatar.UpdatedAt,
	}
}

func wrapCleanupError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func joinUploadError(err error, cleanupErrs ...error) error {
	errs := []error{err}
	for _, cleanupErr := range cleanupErrs {
		if cleanupErr != nil {
			errs = append(errs, cleanupErr)
		}
	}
	return errors.Join(errs...)
}
