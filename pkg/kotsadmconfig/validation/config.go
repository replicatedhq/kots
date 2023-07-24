package validation

import (
	"github.com/pkg/errors"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
)

func ValidateConfigSpec(configSpec kotsv1beta1.ConfigSpec) ([]configtypes.ConfigGroupValidationError, error) {
	var configGroupErrors []configtypes.ConfigGroupValidationError
	for _, configGroup := range configSpec.Groups {
		configGroupError, err := validateConfigGroup(configGroup)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to validate config group %s", configGroup.Name)
		}
		if configGroupError != nil {
			configGroupErrors = append(configGroupErrors, *configGroupError)
		}
	}
	return configGroupErrors, nil
}

func validateConfigGroup(configGroup kotsv1beta1.ConfigGroup) (*configtypes.ConfigGroupValidationError, error) {
	if !isValidatableConfigGroup(configGroup) {
		return nil, nil
	}

	configItemErrors, err := validateConfigItems(configGroup.Items)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate config items")
	}
	if len(configItemErrors) == 0 {
		return nil, nil
	}

	return &configtypes.ConfigGroupValidationError{
		Name:       configGroup.Name,
		Title:      configGroup.Title,
		ItemErrors: configItemErrors,
	}, nil
}

func validateConfigItems(configItems []kotsv1beta1.ConfigItem) ([]configtypes.ConfigItemValidationError, error) {
	var configItemErrors []configtypes.ConfigItemValidationError
	for _, item := range configItems {
		configItemErr, err := validateConfigItem(item)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to validate config item %s", item.Name)
		}

		if configItemErr != nil {
			configItemErrors = append(configItemErrors, *configItemErr)
		}
	}

	return configItemErrors, nil
}

func validateConfigItem(item kotsv1beta1.ConfigItem) (*configtypes.ConfigItemValidationError, error) {
	if !isValidatableConfigItem(item) {
		return nil, nil
	}

	validatableValue, err := getValidatableItemValue(item.Value, item.Type)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validatable value")
	}

	if validatableValue == "" {
		return nil, nil
	}

	validationErrors, err := validate(validatableValue, *item.Validation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate value")
	}

	if len(validationErrors) > 0 {
		return &configtypes.ConfigItemValidationError{
			Name:             item.Name,
			Type:             item.Type,
			ValidationErrors: validationErrors,
		}, nil
	}

	return nil, nil
}

func getValidatableItemValue(value multitype.BoolOrString, itemType string) (string, error) {
	switch itemType {
	case configtypes.TextItemType, configtypes.TextAreaItemType, configtypes.EmptyItemType:
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
		return "", errors.Errorf("item value of type %s validation is not supported", itemType)
	}
}
