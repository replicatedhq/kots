package libyaml

type ContainerSupportCommand struct {
	Filename string   `yaml:"filename" json:"filename" validate:"required"`
	Command  []string `yaml:"command" json:"command"`
}
