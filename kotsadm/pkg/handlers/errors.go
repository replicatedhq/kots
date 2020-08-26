package handlers

type ErrorResponse struct {
	Error   string `json:"error"`
	Success bool   `json:"success"` // NOTE: the frontend relies on this for some routes
}
