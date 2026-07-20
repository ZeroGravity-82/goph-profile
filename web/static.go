package web

import _ "embed"

// DefaultAvatarPNG содержит стандартную PNG-заглушку.
//
//go:embed static/default-avatar.png
var DefaultAvatarPNG []byte

// DefaultAvatar100PNG содержит миниатюру размером 100x100 для стандартной PNG-заглушки.
//
//go:embed static/default-avatar-100.png
var DefaultAvatar100PNG []byte

// DefaultAvatar300PNG содержит миниатюру размером 300x300 для стандартной PNG-заглушки.
//
//go:embed static/default-avatar-300.png
var DefaultAvatar300PNG []byte
