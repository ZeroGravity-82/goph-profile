package worker

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

type avatarWorkerUseCaseFake struct {
	readyErr      error
	failedErr     error
	deletedErr    error
	readyInputs   []usecase.MarkAvatarReadyInput
	failedInputs  []usecase.MarkAvatarFailedInput
	deletedInputs []usecase.MarkAvatarDeletedInput
}

func (uc *avatarWorkerUseCaseFake) MarkAvatarReady(
	_ context.Context,
	in usecase.MarkAvatarReadyInput,
) (usecase.MarkAvatarReadyOutput, error) {
	uc.readyInputs = append(uc.readyInputs, in)
	return usecase.MarkAvatarReadyOutput{Status: model.AvatarStatusReady}, uc.readyErr
}

func (uc *avatarWorkerUseCaseFake) MarkAvatarFailed(
	_ context.Context,
	in usecase.MarkAvatarFailedInput,
) (usecase.MarkAvatarFailedOutput, error) {
	uc.failedInputs = append(uc.failedInputs, in)
	return usecase.MarkAvatarFailedOutput{Status: model.AvatarStatusFailed}, uc.failedErr
}

func (uc *avatarWorkerUseCaseFake) MarkAvatarDeleted(_ context.Context, in usecase.MarkAvatarDeletedInput) error {
	uc.deletedInputs = append(uc.deletedInputs, in)
	return uc.deletedErr
}

type putCall struct {
	objectKey string
	content   []byte
}

type fileStorageFake struct {
	content   []byte
	getErr    error
	putErr    error
	deleteErr error
	gets      []string
	puts      []putCall
	deletes   []string
}

func (s *fileStorageFake) ObjectKeyThumb100(uuid.UUID, uuid.UUID) string {
	return testObjectKeyThumb100
}

func (s *fileStorageFake) ObjectKeyThumb300(uuid.UUID, uuid.UUID) string {
	return testObjectKeyThumb300
}

func (s *fileStorageFake) Get(_ context.Context, objectKey string) ([]byte, error) {
	s.gets = append(s.gets, objectKey)
	return s.content, s.getErr
}

func (s *fileStorageFake) Put(_ context.Context, objectKey string, content []byte) error {
	s.puts = append(s.puts, putCall{objectKey: objectKey, content: content})
	return s.putErr
}

func (s *fileStorageFake) Delete(_ context.Context, objectKey string) error {
	s.deletes = append(s.deletes, objectKey)
	return s.deleteErr
}

func (s *fileStorageFake) putKeys() []string {
	keys := make([]string, 0, len(s.puts))
	for _, put := range s.puts {
		keys = append(keys, put.objectKey)
	}
	return keys
}
