package dto

// LivenessResponse описывает JSON-ответ ручки жизнеспособности сервиса.
type LivenessResponse struct {
	Status string `json:"status"`
}

// ReadinessResponse описывает JSON-ответ ручки готовности сервиса.
type ReadinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}
