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

// DeleteCurrentAvatarInput описывает входные данные сценария удаления текущей аватарки.
type DeleteCurrentAvatarInput struct {
	UserID uuid.UUID
}

// DeleteAvatarInput описывает входные данные сценария удаления аватарки пользователя.
type DeleteAvatarInput struct {
	UserID   uuid.UUID
	AvatarID uuid.UUID
}

// ListUserAvatarsInput описывает входные данные сценария получения списка аватарок пользователя.
type ListUserAvatarsInput struct {
	UserID uuid.UUID
}

// ListUserAvatarsItemOutput описывает элемент в результате сценария получения списка аватарок пользователя.
type ListUserAvatarsItemOutput struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FileName  string
	MIMEType  string
	SizeBytes int64
	Width     *int
	Height    *int
	Status    model.AvatarStatus
	IsCurrent bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListUserAvatarsOutput описывает результат сценария получения списка аватарок пользователя.
type ListUserAvatarsOutput struct {
	Avatars []ListUserAvatarsItemOutput
}

// MarkAvatarReadyInput описывает входные данные сценария завершения обработки аватарки.
type MarkAvatarReadyInput struct {
	AvatarID          uuid.UUID
	Width             int
	Height            int
	ObjectKeyThumb100 string
	ObjectKeyThumb300 string
}

// MarkAvatarReadyOutput описывает результат сценария завершения обработки аватарки.
type MarkAvatarReadyOutput struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Width             int
	Height            int
	ObjectKeyThumb100 string
	ObjectKeyThumb300 string
	Status            model.AvatarStatus
	UpdatedAt         time.Time
}

// MarkAvatarFailedInput описывает входные данные сценария ошибки обработки аватарки.
type MarkAvatarFailedInput struct {
	AvatarID uuid.UUID
}

// MarkAvatarFailedOutput описывает результат сценария ошибки обработки аватарки.
type MarkAvatarFailedOutput struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Status    model.AvatarStatus
	UpdatedAt time.Time
}

// AvatarSize описывает размер файла аватарки, который нужно выдать клиенту.
type AvatarSize string

const (
	// AvatarSizeOriginal означает исходный файл аватарки.
	AvatarSizeOriginal AvatarSize = "original"
	// AvatarSize100 означает миниатюру 100x100.
	AvatarSize100 AvatarSize = "100x100"
	// AvatarSize300 означает миниатюру 300x300.
	AvatarSize300 AvatarSize = "300x300"
)

// GetAvatarInput описывает входные данные сценария получения файла аватарки.
type GetAvatarInput struct {
	AvatarID uuid.UUID
	Size     AvatarSize
	MIMEType string
}

// GetAvatarOutput описывает результат сценария получения файла аватарки.
type GetAvatarOutput struct {
	Content  []byte
	MIMEType string
}

// GetAvatarMetadataInput описывает входные данные сценария получения метаданных аватарки.
type GetAvatarMetadataInput struct {
	AvatarID uuid.UUID
}

// GetAvatarMetadataOutput описывает результат сценария получения метаданных аватарки.
type GetAvatarMetadataOutput struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	FileName          string
	MIMEType          string
	SizeBytes         int64
	Width             *int
	Height            *int
	ObjectKeyOriginal string
	ObjectKeyThumb100 *string
	ObjectKeyThumb300 *string
	Status            model.AvatarStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// GetCurrentAvatarByEmailInput описывает входные данные сценария получения текущей аватарки по email.
type GetCurrentAvatarByEmailInput struct {
	Email model.Email
}

// GetCurrentAvatarByEmailOutput описывает результат сценария получения текущей аватарки по email.
type GetCurrentAvatarByEmailOutput struct {
	Content          []byte
	MIMEType         string
	UseDefaultAvatar bool
}

// avatarUserRepository описывает операции с пользователем, которые нужны сценариям работы с аватарками.
type avatarUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.User, error)
	GetByEmail(ctx context.Context, email model.Email) (model.User, error)
	Update(ctx context.Context, user model.User) error
}

// avatarRepository описывает нужные операции с аватарками.
type avatarRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Avatar, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Avatar, error)
	Create(ctx context.Context, avatar model.Avatar) error
	Update(ctx context.Context, avatar model.Avatar) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// fileStorage описывает операции с хранилищем файлов.
type fileStorage interface {
	ObjectKey(userID uuid.UUID, avatarID uuid.UUID) string
	Get(ctx context.Context, objectKey string) ([]byte, error)
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

// DeleteAvatar удаляет аватарку пользователя.
func (uc *AvatarUseCase) DeleteAvatar(ctx context.Context, in DeleteAvatarInput) error {
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

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return fmt.Errorf("get avatar by id: %w", err)
	}
	if avatar.UserID != user.ID {
		return model.ErrAvatarForbidden
	}

	now := time.Now().UTC()
	if err = avatar.MarkDeleting(now); err != nil {
		return err
	}
	clearCurrent := user.ClearCurrentAvatar(avatar.ID, now)

	if err = uc.avatarRepo.Update(ctx, avatar); err != nil {
		return fmt.Errorf("update deleting avatar: %w", err)
	}
	if clearCurrent {
		if err = uc.userRepo.Update(ctx, user); err != nil {
			return fmt.Errorf("clear current avatar: %w", err)
		}
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

// ListUserAvatars возвращает список неудаленных аватарок пользователя.
func (uc *AvatarUseCase) ListUserAvatars(
	ctx context.Context,
	in ListUserAvatarsInput,
) (ListUserAvatarsOutput, error) {
	if uc.userRepo == nil {
		return ListUserAvatarsOutput{}, errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return ListUserAvatarsOutput{}, errors.New("avatar repository is not provided")
	}

	user, err := uc.userRepo.GetByID(ctx, in.UserID)
	if err != nil {
		return ListUserAvatarsOutput{}, fmt.Errorf("get user by id: %w", err)
	}

	avatars, err := uc.avatarRepo.ListByUserID(ctx, in.UserID)
	if err != nil {
		return ListUserAvatarsOutput{}, fmt.Errorf("list user avatars: %w", err)
	}

	result := ListUserAvatarsOutput{Avatars: make([]ListUserAvatarsItemOutput, 0, len(avatars))}
	for _, avatar := range avatars {
		if avatar.UserID != user.ID || !avatarVisibleInList(avatar) {
			continue
		}
		result.Avatars = append(result.Avatars, listUserAvatarsItemOutput(avatar, user.CurrentAvatarID))
	}

	return result, nil
}

func avatarVisibleInList(avatar model.Avatar) bool {
	return avatar.DeletedAt == nil &&
		avatar.Status != model.AvatarStatusDeleting &&
		avatar.Status != model.AvatarStatusDeleted
}

func listUserAvatarsItemOutput(avatar model.Avatar, currentAvatarID *uuid.UUID) ListUserAvatarsItemOutput {
	return ListUserAvatarsItemOutput{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		FileName:  avatar.FileName,
		MIMEType:  avatar.MIMEType,
		SizeBytes: avatar.SizeBytes,
		Width:     avatar.Width,
		Height:    avatar.Height,
		Status:    avatar.Status,
		IsCurrent: currentAvatarID != nil && *currentAvatarID == avatar.ID,
		CreatedAt: avatar.CreatedAt,
		UpdatedAt: avatar.UpdatedAt,
	}
}

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

// MarkAvatarFailed завершает обработку аватарки ошибкой.
func (uc *AvatarUseCase) MarkAvatarFailed(
	ctx context.Context,
	in MarkAvatarFailedInput,
) (MarkAvatarFailedOutput, error) {
	if uc.avatarRepo == nil {
		return MarkAvatarFailedOutput{}, errors.New("avatar repository is not provided")
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, in.AvatarID)
	if err != nil {
		return MarkAvatarFailedOutput{}, fmt.Errorf("get avatar by id: %w", err)
	}

	if err = avatar.MarkFailed(time.Now().UTC()); err != nil {
		return MarkAvatarFailedOutput{}, err
	}

	if err = uc.avatarRepo.Update(ctx, avatar); err != nil {
		return MarkAvatarFailedOutput{}, fmt.Errorf("update failed avatar: %w", err)
	}

	return MarkAvatarFailedOutput{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		Status:    avatar.Status,
		UpdatedAt: avatar.UpdatedAt,
	}, nil
}

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

// GetCurrentAvatarByEmail возвращает текущую готовую аватарку пользователя или признак выдачи заглушки.
func (uc *AvatarUseCase) GetCurrentAvatarByEmail(
	ctx context.Context,
	in GetCurrentAvatarByEmailInput,
) (GetCurrentAvatarByEmailOutput, error) {
	if uc.userRepo == nil {
		return GetCurrentAvatarByEmailOutput{}, errors.New("user repository is not provided")
	}
	if uc.avatarRepo == nil {
		return GetCurrentAvatarByEmailOutput{}, errors.New("avatar repository is not provided")
	}
	if uc.fileStorage == nil {
		return GetCurrentAvatarByEmailOutput{}, errors.New("file storage is not provided")
	}
	if err := in.Email.Validate(); err != nil {
		return GetCurrentAvatarByEmailOutput{}, err
	}

	user, err := uc.userRepo.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
		}
		return GetCurrentAvatarByEmailOutput{}, fmt.Errorf("get user by email: %w", err)
	}
	if user.CurrentAvatarID == nil {
		return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
	}

	avatar, err := uc.avatarRepo.GetByID(ctx, *user.CurrentAvatarID)
	if err != nil {
		if errors.Is(err, ErrAvatarNotFound) {
			return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
		}
		return GetCurrentAvatarByEmailOutput{}, fmt.Errorf("get current avatar by id: %w", err)
	}
	if avatar.UserID != user.ID {
		return GetCurrentAvatarByEmailOutput{}, model.ErrAvatarForbidden
	}
	if err = avatar.CanBeCurrent(); err != nil {
		return GetCurrentAvatarByEmailOutput{UseDefaultAvatar: true}, nil
	}

	content, err := uc.fileStorage.Get(ctx, avatar.ObjectKeyOriginal)
	if err != nil {
		return GetCurrentAvatarByEmailOutput{}, fmt.Errorf("get original avatar object: %w", err)
	}

	return GetCurrentAvatarByEmailOutput{
		Content:  content,
		MIMEType: avatar.MIMEType,
	}, nil
}
