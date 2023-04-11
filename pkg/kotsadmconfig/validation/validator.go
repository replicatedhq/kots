package validation

import (
	"fmt"
	"regexp"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

const (
	regexMatchError = "Value does not match regex"
)

func validate(item kotsv1beta1.ConfigItem) *ConfigItemError {
	if item.Validation == nil {
		return nil
	}

	if item.Validation.Regex != "" {
		return validateRegex(item)
	}

	return nil
}

func validateRegex(configItem kotsv1beta1.ConfigItem) *ConfigItemError {
	value := configItem.Value.StrVal
	regexStr := configItem.Validation.Regex

	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return buildValidationItemError(configItem, fmt.Sprintf("Invalid regex: %s", err.Error()))
	}

	matched := regex.MatchString(value)
	if !matched {
		return buildValidationItemError(configItem, regexMatchError)
	}

	return nil
}

func buildValidationItemError(configItem kotsv1beta1.ConfigItem, errorMsg string) *ConfigItemError {
	return &ConfigItemError{
		Name:                   configItem.Name,
		Type:                   configItem.Type,
		Value:                  configItem.Value,
		Validation:             *configItem.Validation,
		ValidationErrorMessage: errorMsg,
	}
}
