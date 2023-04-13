package validation

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
)

const (
	BoolItemType      = "bool"
	FileItemType      = "file"
	TextItemType      = "text"
	LabelItemType     = "label"
	HeadingItemType   = "heading"
	PasswordItemType  = "password"
	TextAreaItemType  = "textarea"
	SelectOneItemType = "select_one"
)

type ConfigGroupValidationError struct {
	Name       string                      `json:"name"`
	Title      string                      `json:"title"`
	ItemErrors []ConfigItemValidationError `json:"item_errors"`
}

type ConfigItemValidationError struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Value            multitype.BoolOrString `json:"value"`
	ValidationErrors []ValidationError      `json:"validation_errors"`
}

type ValidationError struct {
	ValidationErrorMessage string                      `json:"validation_error_message"`
	RegexValidator         *kotsv1beta1.RegexValidator `json:"regex,omitempty"`
}
