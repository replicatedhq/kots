package validation

import (
	"fmt"
	"regexp"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

const (
	regexMatchError = "Value does not match regex"
)

func validate(item kotsv1beta1.ConfigItem) *ConfigItemValidationError {
	if item.Validation == nil {
		return nil
	}

	value := item.Value.StrVal
	var validationErrs []ValidationError
	if item.Validation.Regex != nil {
		validationErr := validateRegex(value, item.Validation.Regex)
		if validationErr != nil {
			validationErrs = append(validationErrs, *validationErr)
		}
	}

	if len(validationErrs) > 0 {
		return &ConfigItemValidationError{
			Name:             item.Name,
			Type:             item.Type,
			Value:            item.Value,
			ValidationErrors: validationErrs,
		}
	}

	return nil
}

func validateRegex(value string, regexValidator *kotsv1beta1.RegexValidator) *ValidationError {
	regex, err := regexp.Compile(regexValidator.Regex)
	if err != nil {
		return &ValidationError{
			ValidationErrorMessage: fmt.Sprintf("failed to compile regex %q: %v", regexValidator.Regex, err),
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
