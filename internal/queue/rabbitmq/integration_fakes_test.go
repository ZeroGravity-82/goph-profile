//go:build integration

package rabbitmq

import (
	"context"

	"github.com/ZeroGravity-82/goph-profile/internal/usecase"
)

type avatarMessageHandlerFake struct {
	processing chan usecase.AvatarProcessingMessage
	deletion   chan usecase.AvatarDeletionMessage
}

func newAvatarMessageHandlerFake() *avatarMessageHandlerFake {
	return &avatarMessageHandlerFake{
		processing: make(chan usecase.AvatarProcessingMessage, 1),
		deletion:   make(chan usecase.AvatarDeletionMessage, 1),
	}
}

func (h *avatarMessageHandlerFake) HandleAvatarProcessing(
	_ context.Context,
	message usecase.AvatarProcessingMessage,
) error {
	h.processing <- message
	return nil
}

func (h *avatarMessageHandlerFake) HandleAvatarDeletion(
	_ context.Context,
	message usecase.AvatarDeletionMessage,
) error {
	h.deletion <- message
	return nil
}
