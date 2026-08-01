package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/imageproc"
	"github.com/ZeroGravity-82/goph-profile/internal/logging"
	"github.com/ZeroGravity-82/goph-profile/internal/observability"
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
	metrics     *observability.AvatarAsyncMetrics
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
	metrics, err := observability.NewAvatarAsyncMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to create avatar async metrics: %w", err)
	}

	return &AvatarHandler{
		useCase:     useCase,
		fileStorage: fileStorage,
		metrics:     metrics,
		logger:      logger.With("component", "worker.avatar_handler"),
	}, nil
}

// avatarProcessingMetric накапливает результат задачи обработки аватарки.
// Значение записывается в метрики при выходе из обработчика сообщения.
type avatarProcessingMetric struct {
	startedAt         time.Time
	status            string
	reason            string
	originalSizeBytes int64
	thumb100SizeBytes int64
	thumb300SizeBytes int64
}

func newAvatarProcessingMetric() avatarProcessingMetric {
	return avatarProcessingMetric{
		startedAt: time.Now(),
		status:    "error",
		reason:    "unknown",
	}
}

func (m *avatarProcessingMetric) fail(reason string) {
	m.reason = reason
}

func (m *avatarProcessingMetric) setOriginal(content []byte) {
	m.originalSizeBytes = int64(len(content))
}

func (m *avatarProcessingMetric) setThumbnails(thumbnails imageproc.AvatarThumbnails) {
	m.thumb100SizeBytes = int64(len(thumbnails.Thumb100))
	m.thumb300SizeBytes = int64(len(thumbnails.Thumb300))
}

func (m *avatarProcessingMetric) markFailed() {
	m.status = "failed"
	m.reason = "invalid_image"
}

func (m *avatarProcessingMetric) skip() {
	m.status = "skipped"
	m.reason = "stale_message"
}

func (m *avatarProcessingMetric) success() {
	m.status = "success"
	m.reason = "ready"
}

func (h *AvatarHandler) recordAvatarProcessingMetric(ctx context.Context, metric avatarProcessingMetric) {
	h.metrics.RecordAvatarProcessing(
		ctx,
		metric.status,
		metric.reason,
		time.Since(metric.startedAt),
		metric.originalSizeBytes,
		metric.thumb100SizeBytes,
		metric.thumb300SizeBytes,
	)
}

// HandleAvatarProcessing создает миниатюры и переводит аватарку в готовое состояние.
func (h *AvatarHandler) HandleAvatarProcessing(
	ctx context.Context,
	message usecase.AvatarProcessingMessage,
) error {
	metric := newAvatarProcessingMetric()
	defer func() {
		h.recordAvatarProcessingMetric(ctx, metric)
	}()

	original, err := h.fileStorage.Get(ctx, message.ObjectKeyOriginal)
	if err != nil {
		metric.fail("get_original")
		return fmt.Errorf("failed to get original avatar object: %w", err)
	}
	metric.setOriginal(original)

	thumbnails, err := imageproc.BuildAvatarThumbnails(original)
	if err != nil {
		metric.fail("invalid_image")
		if err = h.markAvatarFailed(ctx, message.AvatarID, err); err != nil {
			metric.fail("mark_failed")
			return err
		}
		metric.markFailed()
		return nil
	}
	metric.setThumbnails(thumbnails)

	thumb100Key := h.fileStorage.ObjectKeyThumb100(message.UserID, message.AvatarID)
	thumb300Key := h.fileStorage.ObjectKeyThumb300(message.UserID, message.AvatarID)
	if err = h.fileStorage.Put(ctx, thumb100Key, thumbnails.Thumb100); err != nil {
		metric.fail("put_thumb_100")
		return fmt.Errorf("failed to put 100x100 avatar thumbnail: %w", err)
	}
	if err = h.fileStorage.Put(ctx, thumb300Key, thumbnails.Thumb300); err != nil {
		metric.fail("put_thumb_300")
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
			metric.skip()
			return nil
		}
		metric.fail("mark_ready")
		return fmt.Errorf("failed to mark avatar ready: %w", err)
	}

	metric.success()
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

// avatarDeletionMetric накапливает результат задачи удаления файлов аватарки.
// Значение записывается в метрики при выходе из обработчика сообщения.
type avatarDeletionMetric struct {
	startedAt      time.Time
	status         string
	reason         string
	deletedObjects int64
}

func newAvatarDeletionMetric() avatarDeletionMetric {
	return avatarDeletionMetric{
		startedAt: time.Now(),
		status:    "error",
		reason:    "unknown",
	}
}

func (m *avatarDeletionMetric) fail(reason string) {
	m.reason = reason
}

func (m *avatarDeletionMetric) incrementDeletedObjects() {
	m.deletedObjects++
}

func (m *avatarDeletionMetric) skip() {
	m.status = "skipped"
	m.reason = "stale_message"
}

func (m *avatarDeletionMetric) success() {
	m.status = "success"
	m.reason = "deleted"
}

func (h *AvatarHandler) recordAvatarDeletionMetric(ctx context.Context, metric avatarDeletionMetric) {
	h.metrics.RecordAvatarDeletion(ctx, metric.status, metric.reason, time.Since(metric.startedAt), metric.deletedObjects)
}

// HandleAvatarDeletion удаляет файлы аватарки и завершает удаление в usecase.
func (h *AvatarHandler) HandleAvatarDeletion(ctx context.Context, message usecase.AvatarDeletionMessage) error {
	metric := newAvatarDeletionMetric()
	defer func() {
		h.recordAvatarDeletionMetric(ctx, metric)
	}()

	for _, objectKey := range message.ObjectKeys {
		if err := h.fileStorage.Delete(ctx, objectKey); err != nil {
			metric.fail("delete_object")
			return fmt.Errorf("failed to delete avatar object: %w", err)
		}
		metric.incrementDeletedObjects()
	}

	if err := h.useCase.MarkAvatarDeleted(ctx, usecase.MarkAvatarDeletedInput{AvatarID: message.AvatarID}); err != nil {
		if errors.Is(err, repository.ErrAvatarNotFound) || errors.Is(err, model.ErrInvalidAvatarTransition) {
			metric.skip()
			return nil
		}
		metric.fail("mark_deleted")
		return fmt.Errorf("failed to mark avatar deleted: %w", err)
	}

	metric.success()
	return nil
}
