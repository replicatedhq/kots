package validation

import (
	"regexp"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
)

const (
	regexMatchError = "Value does not match regex"
)

type regexValidator struct {
	*kotsv1beta1.RegexValidator
}

func (v *regexValidator) Validate(input string) (*configtypes.ValidationError, error) {
	regex, err := regexp.Compile(v.Pattern)
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid regex pattern. failed to compile regex")
	}

	matched := regex.MatchString(input)
	if !matched {
		return buildRegexValidationError(v.Message, regexMatchError), nil
	}
	return nil, nil
}

func buildRegexValidationError(validatorMessage string, errorMsg string) *configtypes.ValidationError {
	if validatorMessage == "" {
		validatorMessage = regexMatchError
	}
	return &configtypes.ValidationError{
		Reason:  errorMsg,
		Message: validatorMessage,
	}
}
