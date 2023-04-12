package validation

import (
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/util"
)

func ValidateConfigSpec(configSpec kotsv1beta1.ConfigSpec) []ConfigGroupValidationError {
	if !hasConfigItemValidators(configSpec) {
		return nil
	}

	var configGroupErrors []ConfigGroupValidationError
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

func validateConfigGroup(configGroup kotsv1beta1.ConfigGroup) *ConfigGroupValidationError {
	configItemErrors := validateConfigItems(configGroup.Items)
	if len(configItemErrors) == 0 {
		return nil
	}

	return &ConfigGroupValidationError{
		Name:       configGroup.Name,
		Title:      configGroup.Title,
		ItemErrors: configItemErrors,
	}
}

func validateConfigItems(configItems []kotsv1beta1.ConfigItem) []ConfigItemValidationError {
	var configItemErrors []ConfigItemValidationError
	for _, item := range configItems {
		configItemErr := validateConfigItem(item)
		if configItemErr != nil {
			configItemErrors = append(configItemErrors, *configItemErr)
		}
	}
	return configItemErrors
}

func validateConfigItem(item kotsv1beta1.ConfigItem) *ConfigItemValidationError {
	if !isValidatableConfigItem(item) {
		return nil
	}

	value, err := getItemValue(item.Value, item.Type)
	if err != nil {
		return &ConfigItemValidationError{
			Name:  item.Name,
			Type:  item.Type,
			Value: item.Value,
			ValidationErrors: []ValidationError{
				{
					ValidationErrorMessage: errors.Wrapf(err, "failed to get item value").Error(),
				},
			},
		}
	}

	validationErrors := validate(value, *item.Validation)
	if len(validationErrors) > 0 {
		return &ConfigItemValidationError{
			Name:             item.Name,
			Type:             item.Type,
			Value:            item.Value,
			ValidationErrors: validationErrors,
		}
	}

	return nil
}

func getItemValue(value multitype.BoolOrString, itemType string) (string, error) {
	switch itemType {
	case TextItemType, TextAreaItemType:
		return value.StrVal, nil
	case PasswordItemType:
		// if decrypting succeeds, use the decrypted value
		if updatedValue, err := util.DecryptConfigValue(value.String()); err == nil {
			return updatedValue, nil
		} else {
			return value.String(), nil
		}
	case FileItemType:
		decodedBytes, err := util.Base64DecodeInterface(value.StrVal)
		if err != nil {
			return "", errors.Wrapf(err, "failed to base64 decode file item value")
		}
		return string(decodedBytes), err
	case HeadingItemType, LabelItemType:
		return "", nil
	default:
		return "", errors.Errorf("unknown item type %s", itemType)
	}
}
