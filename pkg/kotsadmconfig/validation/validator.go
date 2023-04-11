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
	if item.When == "false" || item.Validation == nil {
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
		return buildConfigItemError(configItem, fmt.Sprintf("Invalid regex: %s", err.Error()))
	}

	matched := regex.MatchString(value)
	if !matched {
		return buildConfigItemError(configItem, regexMatchError)
	}

	return nil
}
