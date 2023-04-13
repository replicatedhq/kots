package validation

import (
	"fmt"
	"regexp"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
)

const (
	regexMatchError = "Value does not match regex"
)

type regexValidator struct {
	*kotsv1beta1.RegexValidator
}

func (v *regexValidator) Validate(input string) *configtypes.ValidationError {
	regex, err := regexp.Compile(v.Pattern)
	if err != nil {
		return buildRegexValidationError(v.Message, fmt.Sprintf("Invalid regex: %s", err.Error()))
	}

	matched := regex.MatchString(input)
	if !matched {
		return buildRegexValidationError(v.Message, regexMatchError)
	}
	return nil
}

func buildRegexValidationError(validatorMessage string, errorMsg string) *configtypes.ValidationError {
	if validatorMessage == "" {
		validatorMessage = regexMatchError
	}
	return &configtypes.ValidationError{
		Error:   errorMsg,
		Message: validatorMessage,
	}
}
