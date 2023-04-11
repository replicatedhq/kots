package validation

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
)

type ConfigGroupError struct {
	Name   string            `json:"name"`
	Title  string            `json:"title"`
	Errors []ConfigItemError `json:"errors"`
}

type ConfigItemError struct {
	Name                   string                           `json:"name"`
	Type                   string                           `json:"type"`
	Value                  multitype.BoolOrString           `json:"value"`
	ValidationErrorMessage string                           `json:"validation_error_message"`
	Validation             kotsv1beta1.ConfigItemValidation `json:"validation"`
	ChildItemErrors        []ConfigItemError                `json:"child_item_errors,omitempty"`
}
