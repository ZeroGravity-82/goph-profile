package web

import _ "embed"

// DefaultAvatarPNG содержит стандартную аватарку, встроенную в бинарник приложения.
//
//go:embed static/default-avatar.png
var DefaultAvatarPNG []byte
