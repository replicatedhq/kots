package validation

import (
	"regexp"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

const (
	RegexValidationType = "regex"
)

func validate(item kotsv1beta1.ConfigItem) *ConfigItemError {
	if item.Validation == nil {
		return nil
	}

	switch item.Validation.Type {
	case RegexValidationType:
		return validateRegex(item)
	default:
		return nil
	}
}

func validateRegex(configItem kotsv1beta1.ConfigItem) *ConfigItemError {
	matched, err := regexp.MatchString(configItem.Validation.Rule, configItem.Value.String())
	if err != nil {
		return buildValidationItemError(configItem, err.Error())
	}

	if !matched {
		return buildValidationItemError(configItem, "Value does not match regex")
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
