package libyaml

// ConfigChildItem represents a subitem.  This is normally found in select_one and select_many types.
type ConfigChildItem struct {
	Name        string `yaml:"name" json:"name" validate:"required"`
	Title       string `yaml:"title" json:"title"`
	Recommended bool   `yaml:"recommended" json:"recommended"`
	Default     string `yaml:"default" json:"default"`
	Value       string `yaml:"value" json:"value"`
}
