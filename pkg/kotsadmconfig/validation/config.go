package validation

import (
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

func HasConfigItemValidators(config kotsv1beta1.Config) bool {
	for _, configGroup := range config.Spec.Groups {
		for _, configItem := range configGroup.Items {
			if configItem.Validation != nil {
				return true
			}

			for _, configChildItem := range configItem.Items {
				if configChildItem.Validation != nil {
					return true
				}
			}
		}
	}

	return false
}

func ValidateConfigGroups(configGroups []kotsv1beta1.ConfigGroup) []ConfigGroupError {
	var configGroupErrors []ConfigGroupError
	for _, configGroup := range configGroups {
		configGroupError := validateConfigGroup(configGroup)
		if configGroupError != nil {
			configGroupErrors = append(configGroupErrors, *configGroupError)
		}
	}
	return configGroupErrors
}

func validateConfigGroup(configGroup kotsv1beta1.ConfigGroup) *ConfigGroupError {
	configItemErrors := validateConfigItems(configGroup.Items)
	if len(configItemErrors) == 0 {
		return nil
	}

	return &ConfigGroupError{
		Name:   configGroup.Name,
		Title:  configGroup.Title,
		Errors: configItemErrors,
	}
}

func validateConfigItems(configItems []kotsv1beta1.ConfigItem) []ConfigItemError {
	var configItemErrors []ConfigItemError
	for _, configItem := range configItems {
		// validate config item
		congigItemErr := validate(configItem)

		// validate configChildItems
		configChildItemErrors := validateConfigChildItems(configItem.Items)
		if len(configChildItemErrors) > 0 {
			if congigItemErr == nil {
				congigItemErr = &ConfigItemError{
					Name: configItem.Name,
					Type: configItem.Type,
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

func validateConfigChildItems(configChildItems []kotsv1beta1.ConfigChildItem) []ConfigItemError {
	var configChildItemErrors []ConfigItemError
	for _, configChildItem := range configChildItems {
		validateConfigChildItemErr := validateConfigChildItem(configChildItem)
		if validateConfigChildItemErr != nil {
			configChildItemErrors = append(configChildItemErrors, *validateConfigChildItemErr)
		}
	}
	return configChildItemErrors
}

func validateConfigChildItem(childConfigItem kotsv1beta1.ConfigChildItem) *ConfigItemError {
	if childConfigItem.Validation == nil {
		return nil
	}

	configItem := kotsv1beta1.ConfigItem{
		Name:       childConfigItem.Name,
		Value:      childConfigItem.Value,
		Validation: childConfigItem.Validation,
	}
	return validate(configItem)
}
