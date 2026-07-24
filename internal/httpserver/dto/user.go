package dto

// ResolveUserRequest описывает JSON-запрос определения пользователя по email.
type ResolveUserRequest struct {
	Email string `json:"email"`
}

// ResolveUserResponse описывает JSON-ответ определения пользователя по email.
type ResolveUserResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}
