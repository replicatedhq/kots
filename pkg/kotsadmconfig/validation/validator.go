package validation

import (
	"github.com/pkg/errors"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

var (
	validatableItemTypesMap = map[string]bool{
		configtypes.EmptyItemType:    true,
		configtypes.TextItemType:     true,
		configtypes.PasswordItemType: true,
		configtypes.TextAreaItemType: true,
		configtypes.FileItemType:     true,
	}
)

func isValidatableConfigGroup(group kotsv1beta1.ConfigGroup) bool {
	if group.When == "false" {
		return false
	}

	if len(group.Items) == 0 {
		return false
	}

	return true
}

func isValidatableConfigItem(item kotsv1beta1.ConfigItem) bool {
	if item.Validation == nil {
		return false
	}

	if item.Hidden {
		return false
	}

	if item.When == "false" {
		return false
	}

	if item.Repeatable {
		return false
	}

	if !validatableItemTypesMap[item.Type] {
		return false
	}

	return true
}

func validate(value string, itemValidation kotsv1beta1.ConfigItemValidation) ([]configtypes.ValidationError, error) {
	var validationErrs []configtypes.ValidationError
	validators := buildValidators(itemValidation)
	for _, v := range validators {
		validationErr, err := v.Validate(value)
		if err != nil {
			return nil, errors.Wrap(err, "failed to validate")
		}

		if validationErr != nil {
			validationErrs = append(validationErrs, *validationErr)
		}
	}

	return validationErrs, nil
}

func buildValidators(itemValidator kotsv1beta1.ConfigItemValidation) []validator {
	var validators []validator
	if itemValidator.Regex != nil {
		validators = append(validators, &regexValidator{itemValidator.Regex})
	}
	return validators
}
