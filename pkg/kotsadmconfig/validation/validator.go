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
	return item.Validation != nil && !item.Hidden && item.When != "false" && !item.Repeatable && validatableItemTypesMap[item.Type]
}

func validate(value string, itemValidator kotsv1beta1.ConfigItemValidation) []configtypes.ValidationError {
	var validationErrs []configtypes.ValidationError
	validators := buildValidators(itemValidator)
	for _, v := range validators {
		validationErr := v.Validate(value)
		if validationErr != nil {
			validationErrs = append(validationErrs, *validationErr)
		}
	}

	return validationErrs
}

func buildValidators(itemValidator kotsv1beta1.ConfigItemValidation) []validator {
	var validators []validator
	if itemValidator.Regex != nil {
		validators = append(validators, &regexValidator{itemValidator.Regex})
	}
	return validators
}
