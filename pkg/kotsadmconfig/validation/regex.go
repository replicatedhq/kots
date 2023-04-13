package validation

import (
	"fmt"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"regexp"
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
		return &configtypes.ValidationError{
			ValidationErrorMessage: fmt.Sprintf("Invalid regex: %s", err.Error()),
			RegexValidator:         v.RegexValidator,
		}
	}

	matched := regex.MatchString(input)
	if !matched {
		return &configtypes.ValidationError{
			ValidationErrorMessage: regexMatchError,
			RegexValidator:         v.RegexValidator,
		}
	}
	return nil
}
