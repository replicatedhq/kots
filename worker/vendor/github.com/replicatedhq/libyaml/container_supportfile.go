package libyaml

type ContainerSupportFile struct {
	Filename string `yaml:"filename" json:"filename" validate:"required"`
}
