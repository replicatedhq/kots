package types

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

type ErrorResponse struct {
	Error   *string `json:"error,omitempty"`
	Success bool    `json:"success"` // NOTE: the frontend relies on this for some routes
	Err     error   `json:"-"`
}

func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Error:   util.StrPointer(errors.Cause(err).Error()),
		Success: false,
		Err:     err,
	}
}

func ErrorFromResponse(body []byte) string {
	errorResponse := ErrorResponse{}
	json.Unmarshal(body, &errorResponse)
	if errorResponse.Error == nil {
		return string(body)
	}
	return *errorResponse.Error
}
