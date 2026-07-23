package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/imageproc"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/repository"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

type avatarWorkerUseCase interface {
	MarkAvatarReady(ctx context.Context, in usecase.MarkAvatarReadyInput) (usecase.MarkAvatarReadyOutput, error)
	MarkAvatarFailed(ctx context.Context, in usecase.MarkAvatarFailedInput) (usecase.MarkAvatarFailedOutput, error)
	MarkAvatarDeleted(ctx context.Context, in usecase.MarkAvatarDeletedInput) error
}

type fileStorage interface {
	ObjectKeyThumb100(userID uuid.UUID, avatarID uuid.UUID) string
	ObjectKeyThumb300(userID uuid.UUID, avatarID uuid.UUID) string
	Get(ctx context.Context, objectKey string) ([]byte, error)
	Put(ctx context.Context, objectKey string, content []byte) error
	Delete(ctx context.Context, objectKey string) error
}

// AvatarHandler обрабатывает сообщения воркера по аватаркам.
type AvatarHandler struct {
	useCase     avatarWorkerUseCase
	fileStorage fileStorage
	logger      *slog.Logger
}

// NewAvatarHandler создает AvatarHandler.
func NewAvatarHandler(
	useCase avatarWorkerUseCase,
	fileStorage fileStorage,
	logger *slog.Logger,
) (*AvatarHandler, error) {
	if useCase == nil {
		return nil, errors.New("avatar worker use case is not provided")
	}
	if fileStorage == nil {
		return nil, errors.New("file storage is not provided")
	}
	if logger == nil {
		logger = logging.NopLogger()
	}

	return &AvatarHandler{useCase: useCase, fileStorage: fileStorage, logger: logger}, nil
}

// HandleAvatarProcessing создает миниатюры и переводит аватарку в готовое состояние.
func (h *AvatarHandler) HandleAvatarProcessing(
	ctx context.Context,
	message usecase.AvatarProcessingMessage,
) error {
	original, err := h.fileStorage.Get(ctx, message.ObjectKeyOriginal)
	if err != nil {
		return fmt.Errorf("failed to get original avatar object: %w", err)
	}

	thumbnails, err := imageproc.BuildAvatarThumbnails(original)
	if err != nil {
		return h.markAvatarFailed(ctx, message.AvatarID, err)
	}

	thumb100Key := h.fileStorage.ObjectKeyThumb100(message.UserID, message.AvatarID)
	thumb300Key := h.fileStorage.ObjectKeyThumb300(message.UserID, message.AvatarID)
	if err = h.fileStorage.Put(ctx, thumb100Key, thumbnails.Thumb100); err != nil {
		return fmt.Errorf("failed to put 100x100 avatar thumbnail: %w", err)
	}
	if err = h.fileStorage.Put(ctx, thumb300Key, thumbnails.Thumb300); err != nil {
		return fmt.Errorf("failed to put 300x300 avatar thumbnail: %w", err)
	}

	_, err = h.useCase.MarkAvatarReady(ctx, usecase.MarkAvatarReadyInput{
		AvatarID:          message.AvatarID,
		Width:             thumbnails.Width,
		Height:            thumbnails.Height,
		ObjectKeyThumb100: thumb100Key,
		ObjectKeyThumb300: thumb300Key,
	})
	if err != nil {
		if staleAvatarProcessingMessage(err) {
			return nil
		}
		return fmt.Errorf("failed to mark avatar ready: %w", err)
	}

	return nil
}

// staleAvatarProcessingMessage определяет, что задача обработки уже не соответствует текущему состоянию аватарки.
func staleAvatarProcessingMessage(err error) bool {
	return errors.Is(err, repository.ErrAvatarNotFound) || errors.Is(err, model.ErrInvalidAvatarTransition)
}

func (h *AvatarHandler) markAvatarFailed(ctx context.Context, avatarID uuid.UUID, cause error) error {
	h.logger.WarnContext(ctx, "avatar processing failed",
		slog.String("avatar_id", avatarID.String()),
		slog.Any("err", cause),
	)
	if _, err := h.useCase.MarkAvatarFailed(ctx, usecase.MarkAvatarFailedInput{AvatarID: avatarID}); err != nil {
		if staleAvatarProcessingMessage(err) {
			return nil
		}
		return fmt.Errorf("failed to mark avatar failed: %w", err)
	}
	return nil
}

// HandleAvatarDeletion удаляет файлы аватарки и завершает удаление в usecase.
func (h *AvatarHandler) HandleAvatarDeletion(ctx context.Context, message usecase.AvatarDeletionMessage) error {
	for _, objectKey := range message.ObjectKeys {
		if err := h.fileStorage.Delete(ctx, objectKey); err != nil {
			return fmt.Errorf("failed to delete avatar object: %w", err)
		}
	}

	if err := h.useCase.MarkAvatarDeleted(ctx, usecase.MarkAvatarDeletedInput{AvatarID: message.AvatarID}); err != nil {
		if errors.Is(err, repository.ErrAvatarNotFound) || errors.Is(err, model.ErrInvalidAvatarTransition) {
			return nil
		}
		return fmt.Errorf("failed to mark avatar deleted: %w", err)
	}

	return nil
}
