package dto

// HealthResponse описывает JSON-ответ ручки проверки состояния сервиса.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}
