package validation

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"github.com/replicatedhq/kots/pkg/util"
)

func ValidateConfigSpec(configSpec kotsv1beta1.ConfigSpec) []configtypes.ConfigGroupValidationError {
	if !hasConfigItemValidators(configSpec) {
		return nil
	}

	var configGroupErrors []configtypes.ConfigGroupValidationError
	for _, configGroup := range configSpec.Groups {
		configGroupError := validateConfigGroup(configGroup)
		if configGroupError != nil {
			configGroupErrors = append(configGroupErrors, *configGroupError)
		}
	}
	return configGroupErrors
}

func hasConfigItemValidators(configSpec kotsv1beta1.ConfigSpec) bool {
	for _, configGroup := range configSpec.Groups {
		for _, item := range configGroup.Items {
			if isValidatableConfigItem(item) {
				return true
			}
		}
	}

	return false
}

func validateConfigGroup(configGroup kotsv1beta1.ConfigGroup) *configtypes.ConfigGroupValidationError {
	configItemErrors := validateConfigItems(configGroup.Items)
	if len(configItemErrors) == 0 {
		return nil
	}

	return &configtypes.ConfigGroupValidationError{
		Name:       configGroup.Name,
		Title:      configGroup.Title,
		ItemErrors: configItemErrors,
	}
}

func validateConfigItems(configItems []kotsv1beta1.ConfigItem) []configtypes.ConfigItemValidationError {
	var configItemErrors []configtypes.ConfigItemValidationError
	for _, item := range configItems {
		configItemErr := validateConfigItem(item)
		if configItemErr != nil {
			configItemErrors = append(configItemErrors, *configItemErr)
		}
	}
	return configItemErrors
}

func validateConfigItem(item kotsv1beta1.ConfigItem) *configtypes.ConfigItemValidationError {
	if !isValidatableConfigItem(item) {
		return nil
	}

	value, err := getValidatableItemValue(item.Value, item.Type)
	if err != nil {
		return &configtypes.ConfigItemValidationError{
			Name: item.Name,
			Type: item.Type,
			ValidationErrors: []configtypes.ValidationError{
				{
					Message: errors.Wrapf(err, "failed to get item value").Error(),
				},
			},
		}
	}

	validationErrors := validate(value, *item.Validation)
	if len(validationErrors) > 0 {
		return &configtypes.ConfigItemValidationError{
			Name:             item.Name,
			Type:             item.Type,
			ValidationErrors: validationErrors,
		}
	}

	return nil
}

func getValidatableItemValue(value multitype.BoolOrString, itemType string) (string, error) {
	switch itemType {
	case configtypes.TextItemType, configtypes.TextAreaItemType:
		return value.StrVal, nil
	case configtypes.PasswordItemType:
		// if decrypting succeeds, use the decrypted value
		if updatedValue, err := util.DecryptConfigValue(value.String()); err == nil {
			return updatedValue, nil
		} else {
			return value.String(), nil
		}
	case configtypes.FileItemType:
		decodedBytes, err := util.Base64DecodeInterface(value.StrVal)
		if err != nil {
			return "", errors.Wrapf(err, "failed to base64 decode file item value")
		}
		return string(decodedBytes), err
	default:
		return "", errors.Errorf("item value of type %s is not validated", itemType)
	}
}
