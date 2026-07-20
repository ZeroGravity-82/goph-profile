package usecase

import (
	"context"
	"errors"
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
