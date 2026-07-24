package handler

import (
	"bytes"
	"log/slog"
)

func discardLogger() *slog.Logger {
	return newTextLogger(&bytes.Buffer{})
}

func newTextLogger(buffer *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buffer, nil))
}
