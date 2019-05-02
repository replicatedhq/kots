package libyaml

type ContainerExtraHost struct {
	Hostname string `yaml:"hostname" json:"hostname" validate:"required"`
	Address  string `yaml:"address" json:"address" validate:"required"`
	When     string `yaml:"when" json:"when"`
}
