package health

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}
