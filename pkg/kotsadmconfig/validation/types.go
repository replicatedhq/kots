package validation

type ConfigItemError struct {
	Name 	string `json:"name"`
	Type 	string `json:"type"`
	ValidationType string `json:"validation_type"`
	ValidationMessage string `json:"validation_message"`
	ValidationErrorMessage string `json:"validation_error_message"`

	ChildItemErrors []ConfigItemError `json:"child_item_errors"`
}

// type ConfigItemErrors []ConfigItemError

type ConfigGroupError struct {
	Name 	string `json:"name"`
	Title 	string `json:"title"`
	Errors 	[]ConfigItemError `json:"errors"`
}
