package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/ZeroGravity-82/goph-profile/internal/domain/model"
)

type avatarUserRepositoryFake struct {
	user      model.User
	err       error
	emailErr  error
	updateErr error
	ids       []uuid.UUID
	emails    []model.Email
	updated   []model.User
}

func (r *avatarUserRepositoryFake) GetByID(_ context.Context, id uuid.UUID) (model.User, error) {
	r.ids = append(r.ids, id)
	return r.user, r.err
}

func (r *avatarUserRepositoryFake) GetByEmail(_ context.Context, email model.Email) (model.User, error) {
	r.emails = append(r.emails, email)
	return r.user, r.emailErr
}

func (r *avatarUserRepositoryFake) Update(_ context.Context, user model.User) error {
	r.updated = append(r.updated, user)
	return r.updateErr
}

type avatarRepositoryFake struct {
	avatar     model.Avatar
	avatars    []model.Avatar
	getErr     error
	listErr    error
	createErr  error
	updateErr  error
	deleteErr  error
	ids        []uuid.UUID
	userIDs    []uuid.UUID
	created    []model.Avatar
	updated    []model.Avatar
	deletedIDs []uuid.UUID
}

func (r *avatarRepositoryFake) GetByID(_ context.Context, id uuid.UUID) (model.Avatar, error) {
	r.ids = append(r.ids, id)
	return r.avatar, r.getErr
}

func (r *avatarRepositoryFake) ListByUserID(_ context.Context, userID uuid.UUID) ([]model.Avatar, error) {
	r.userIDs = append(r.userIDs, userID)
	return r.avatars, r.listErr
}

func (r *avatarRepositoryFake) Create(_ context.Context, avatar model.Avatar) error {
	r.created = append(r.created, avatar)
	return r.createErr
}

func (r *avatarRepositoryFake) Update(_ context.Context, avatar model.Avatar) error {
	r.updated = append(r.updated, avatar)
	return r.updateErr
}

func (r *avatarRepositoryFake) Delete(_ context.Context, id uuid.UUID) error {
	r.deletedIDs = append(r.deletedIDs, id)
	return r.deleteErr
}

type transactorFake struct {
	err   error
	calls int
}

func (t *transactorFake) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.calls++
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

type putObjectCall struct {
	objectKey string
	content   []byte
}

type objectKeyCall struct {
	userID   uuid.UUID
	avatarID uuid.UUID
}

type fileStoreFake struct {
	objectKey      string
	content        []byte
	getErr         error
	putErr         error
	deleteErr      error
	objectKeyCalls []objectKeyCall
	gets           []string
	puts           []putObjectCall
	deletes        []string
}

func (s *fileStoreFake) ObjectKey(userID uuid.UUID, avatarID uuid.UUID) string {
	s.objectKeyCalls = append(s.objectKeyCalls, objectKeyCall{userID: userID, avatarID: avatarID})
	if s.objectKey != "" {
		return s.objectKey
	}
	return testObjectKeyOriginal
}

func (s *fileStoreFake) Get(_ context.Context, objectKey string) ([]byte, error) {
	s.gets = append(s.gets, objectKey)
	return s.content, s.getErr
}

func (s *fileStoreFake) Put(_ context.Context, objectKey string, content []byte) error {
	s.puts = append(s.puts, putObjectCall{objectKey: objectKey, content: content})
	return s.putErr
}

func (s *fileStoreFake) Delete(_ context.Context, objectKey string) error {
	s.deletes = append(s.deletes, objectKey)
	return s.deleteErr
}

type avatarMessagePublisherFake struct {
	err              error
	deleteErr        error
	messages         []AvatarProcessingMessage
	deletionMessages []AvatarDeletionMessage
}

func (p *avatarMessagePublisherFake) PublishAvatarProcessing(
	_ context.Context,
	message AvatarProcessingMessage,
) error {
	p.messages = append(p.messages, message)
	return p.err
}

func (p *avatarMessagePublisherFake) PublishAvatarDeletion(
	_ context.Context,
	message AvatarDeletionMessage,
) error {
	p.deletionMessages = append(p.deletionMessages, message)
	return p.deleteErr
}
