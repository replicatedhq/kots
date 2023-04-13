package validation

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
)

var (
	validatableItemTypesMap = map[string]bool{
		configtypes.TextItemType:     true,
		configtypes.PasswordItemType: true,
		configtypes.TextAreaItemType: true,
		configtypes.FileItemType:     true,
	}
)

func isValidatableConfigItem(item kotsv1beta1.ConfigItem) bool {
	if item.Validation == nil ||
		item.Hidden ||
		item.When == "false" ||
		item.Repeatable ||
		!validatableItemTypesMap[item.Type] {
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
			return nil, err
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
