package validation

import (
	"fmt"
	"regexp"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
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

const (
	regexMatchError = "Value does not match regex"
)

var (
	validatableItemTypesMap = map[string]bool{
		TextItemType:     true,
		PasswordItemType: true,
		TextAreaItemType: true,
		FileItemType:     true,
	}
)

func isValidatableConfigItem(item kotsv1beta1.ConfigItem) bool {
	return item.Validation != nil && !item.Hidden && item.When != "false" && !item.Repeatable && validatableItemTypesMap[item.Type]
}

func validate(value string, validator kotsv1beta1.ConfigItemValidator) []ValidationError {
	var validationErrs []ValidationError

	if validator.Regex != nil {
		validationErr := validateRegex(value, validator.Regex)
		if validationErr != nil {
			validationErrs = append(validationErrs, *validationErr)
		}
	}

	return validationErrs
}

func validateRegex(value string, regexValidator *kotsv1beta1.RegexValidator) *ValidationError {
	regex, err := regexp.Compile(regexValidator.Pattern)
	if err != nil {
		return &ValidationError{
			ValidationErrorMessage: fmt.Sprintf("failed to compile regex %q: %v", regexValidator.Pattern, err),
			RegexValidator:         regexValidator,
		}
	}

	matched := regex.MatchString(value)
	if !matched {
		return &ValidationError{
			ValidationErrorMessage: regexMatchError,
			RegexValidator:         regexValidator,
		}
	}
	return nil
}
