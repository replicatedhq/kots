package validation

const (
	EmptyItemType     = "" // when type is not set, it defaults to text
	BoolItemType      = "bool"
	FileItemType      = "file"
	TextItemType      = "text"
	LabelItemType     = "label"
	HeadingItemType   = "heading"
	PasswordItemType  = "password"
	TextAreaItemType  = "textarea"
	SelectOneItemType = "select_one"
	RadioItemType     = "radio"
	DropdownItemType  = "dropdown"
)

type ConfigGroupValidationError struct {
	Name       string                      `json:"name"`
	Title      string                      `json:"title"`
	ItemErrors []ConfigItemValidationError `json:"item_errors"`
}

type ConfigItemValidationError struct {
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	ValidationErrors []ValidationError `json:"validation_errors"`
}

type ValidationError struct {
	Message string `json:"message"`
}
