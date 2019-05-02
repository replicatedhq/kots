package libyaml

import (
	"reflect"

	validator "gopkg.in/go-playground/validator.v8"
)

// CustomRequirementIDUnique will validate that the custom requirement id is unique to all other custom requirements
func CustomRequirementIDUnique(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		// this is an issue with the code and really should be a panic
		return true
	}

	requirementID := field.String()

	root, ok := topStruct.Interface().(*RootConfig)
	if !ok {
		// this is an issue with the code and really should be a panic
		return true
	}

	count := countCustomRequirementsWithID(requirementID, root)
	if count > 1 {
		return false
	}
	return true
}

func countCustomRequirementsWithID(id string, root *RootConfig) int {
	var count int
	for _, customRequirement := range root.CustomRequirements {
		if customRequirement.ID == id {
			count++
		}
	}
	return count
}
