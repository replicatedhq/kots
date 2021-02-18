package handlers

import "github.com/pkg/errors"

type ErrorResponse struct {
	Error   string `json:"error"`
	Success bool   `json:"success"` // NOTE: the frontend relies on this for some routes
	Err     error  `json:"-"`
}

func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Error:   errors.Cause(err).Error(),
		Success: false,
		Err:     err,
	}
}
