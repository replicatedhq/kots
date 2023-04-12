package validation

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
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
			if item.Hidden || item.When == "false" {
				continue
			}

			if item.Validation != nil {
				return true
			}

			for _, configChildItem := range item.Items {
				if configChildItem.Validation != nil {
					return true
				}
			}

		}
	}

	return false
}

func hasConfigItemValidator(item kotsv1beta1.ConfigItem) bool {
	return !item.Hidden && item.When != "false" && item.Validation != nil
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
		if item.Hidden || item.When == "false" {
			continue
		}
		// validate config item
		congigItemErr := validate(item)

		// validate configChildItems
		configChildItemErrors := validateConfigChildItems(item.Items)
		if len(configChildItemErrors) > 0 {
			if congigItemErr == nil {
				congigItemErr = &ConfigItemValidationError{
					Name: item.Name,
					Type: item.Type,
				}
			}
			congigItemErr.ChildItemErrors = configChildItemErrors
		}

		if congigItemErr != nil {
			configItemErrors = append(configItemErrors, *congigItemErr)
		}
	}
	return configItemErrors
}

func validateConfigChildItems(configChildItems []kotsv1beta1.ConfigChildItem) []ConfigItemValidationError {
	var configChildItemErrors []ConfigItemValidationError
	for _, configChildItem := range configChildItems {
		validateConfigChildItemErr := validateConfigChildItem(configChildItem)
		if validateConfigChildItemErr != nil {
			configChildItemErrors = append(configChildItemErrors, *validateConfigChildItemErr)
		}
	}
	return configChildItemErrors
}

func validateConfigChildItem(childConfigItem kotsv1beta1.ConfigChildItem) *ConfigItemValidationError {
	if childConfigItem.Validation == nil {
		return nil
	}

	// convert ConfigChildItem to ConfigItem for validation
	configItem := kotsv1beta1.ConfigItem{
		Name:       childConfigItem.Name,
		Value:      childConfigItem.Value,
		Validation: childConfigItem.Validation,
	}
	return validate(configItem)
}
