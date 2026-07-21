package usecase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testObjectKeyOriginal = "users/user-id/avatars/avatar-id/original"

var (
	testOtherUserID   = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a002")
	testAvatarID      = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a003")
	testOtherAvatarID = uuid.MustParse("018f2f5d-7cc4-7c52-9f2f-3d3f94f8a004")
)

// TestNewAvatarUseCase_RejectsNilDependencies проверяет обязательность зависимостей.
func TestNewAvatarUseCase_RejectsNilDependencies(t *testing.T) {
	validUserRepo := &avatarUserRepositoryFake{}
	validAvatarRepo := &avatarRepositoryFake{}
	validTransactor := &transactorFake{}
	validFileStore := &fileStoreFake{}
	validMessagePublisher := &avatarMessagePublisherFake{}

	tests := []struct {
		name             string
		userRepo         avatarUserRepository
		avatarRepo       avatarRepository
		transactor       transactor
		fileStore        fileStorage
		messagePublisher avatarMessagePublisher
		wantErr          string
	}{
		{
			name:             "user repository",
			avatarRepo:       validAvatarRepo,
			transactor:       validTransactor,
			fileStore:        validFileStore,
			messagePublisher: validMessagePublisher,
			wantErr:          "user repository is not provided",
		},
		{
			name:             "avatar repository",
			userRepo:         validUserRepo,
			transactor:       validTransactor,
			fileStore:        validFileStore,
			messagePublisher: validMessagePublisher,
			wantErr:          "avatar repository is not provided",
		},
		{
			name:             "transactor",
			userRepo:         validUserRepo,
			avatarRepo:       validAvatarRepo,
			fileStore:        validFileStore,
			messagePublisher: validMessagePublisher,
			wantErr:          "transactor is not provided",
		},
		{
			name:             "file storage",
			userRepo:         validUserRepo,
			avatarRepo:       validAvatarRepo,
			transactor:       validTransactor,
			messagePublisher: validMessagePublisher,
			wantErr:          "file storage is not provided",
		},
		{
			name:       "message publisher",
			userRepo:   validUserRepo,
			avatarRepo: validAvatarRepo,
			transactor: validTransactor,
			fileStore:  validFileStore,
			wantErr:    "avatar message publisher is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			useCase, err := NewAvatarUseCase(
				tt.userRepo,
				tt.avatarRepo,
				tt.transactor,
				tt.fileStore,
				tt.messagePublisher,
			)

			// Assert
			require.EqualError(t, err, tt.wantErr)
			assert.Nil(t, useCase)
		})
	}
}
