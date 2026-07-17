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

// SelectCurrentAvatarInput описывает входные данные сценария выбора текущей аватарки.
type SelectCurrentAvatarInput struct {
	UserID   uuid.UUID
	AvatarID uuid.UUID
}

// SelectCurrentAvatarOutput описывает результат сценария выбора текущей аватарки.
type SelectCurrentAvatarOutput struct {
	UserID          uuid.UUID
	CurrentAvatarID uuid.UUID
	UpdatedAt       time.Time
}

// DeleteCurrentAvatarInput описывает входные данные сценария удаления текущей аватарки.
type DeleteCurrentAvatarInput struct {
	UserID uuid.UUID
}

// avatarUserRepository описывает операции с пользователем, которые нужны сценариям работы с аватарками.
type avatarUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.User, error)
	Update(ctx context.Context, user model.User) error
}

// avatarRepository описывает нужные операции с аватарками.
type avatarRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Avatar, error)
	Create(ctx context.Context, avatar model.Avatar) error
	Update(ctx context.Context, avatar model.Avatar) error
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

// AvatarDeletionMessage описывает сообщение удаления файлов аватарки.
type AvatarDeletionMessage struct {
	AvatarID   uuid.UUID
	ObjectKeys []string
}

// avatarMessagePublisher описывает публикацию сообщений по аватаркам.
type avatarMessagePublisher interface {
	PublishAvatarProcessing(ctx context.Context, message AvatarProcessingMessage) error
	PublishAvatarDeletion(ctx context.Context, message AvatarDeletionMessage) error
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

// SelectCurrentAvatar выбирает готовую аватарку пользователя как текущую.
func (uc *AvatarUseCase) SelectCurrentAvatar(
	ctx context.Context,
	in SelectCurrentAvatarInput,
) (SelectCurrentAvatarOutput, error) {
	if uc == nil || uc.userRepo == nil {
		return SelectCurrentAvatarOutput{}, errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return SelectCurrentAvatarOutput{}, errors.New("avatar repository is not provided")
	}

	user, err := uc.userRepo.GetByID(ctx, in.UserID)
	if err != nil {
		return SelectCurrentAvatarOutput{}, fmt.Errorf("get user by id: %w", err)
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return SelectCurrentAvatarOutput{}, fmt.Errorf("get avatar by id: %w", err)
	}

	alreadyCurrent := user.CurrentAvatarID != nil && *user.CurrentAvatarID == avatar.ID
	if err = user.SelectCurrentAvatar(avatar, time.Now().UTC()); err != nil {
		return SelectCurrentAvatarOutput{}, err
	}
	if alreadyCurrent {
		return selectCurrentAvatarOutput(user), nil
	}

	if err = uc.userRepo.Update(ctx, user); err != nil {
		return SelectCurrentAvatarOutput{}, fmt.Errorf("update current avatar: %w", err)
	}

	return selectCurrentAvatarOutput(user), nil
}

// DeleteCurrentAvatar удаляет текущую аватарку пользователя.
func (uc *AvatarUseCase) DeleteCurrentAvatar(ctx context.Context, in DeleteCurrentAvatarInput) error {
	if uc == nil || uc.userRepo == nil {
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

func selectCurrentAvatarOutput(user model.User) SelectCurrentAvatarOutput {
	return SelectCurrentAvatarOutput{
		UserID:          user.ID,
		CurrentAvatarID: *user.CurrentAvatarID,
		UpdatedAt:       user.UpdatedAt,
	}
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
