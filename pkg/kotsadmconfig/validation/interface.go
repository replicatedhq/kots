package validation

import (
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
)

type validator interface {
	Validate(input string) (*configtypes.ValidationError, error)
}
