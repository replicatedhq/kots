package libyaml

type ContainerPort struct {
	PrivatePort string `yaml:"private_port" json:"private_port" validate:"required"`
	PublicPort  string `yaml:"public_port,omitempty" json:"public_port,omitempty" validate:"required_minapiversion=2.8.0"`
	Interface   string `yaml:"interface,omitempty" json:"interface,omitempty"`
	PortType    string `yaml:"port_type,omitempty" json:"port_type,omitempty"`
	When        string `yaml:"when,omitempty" json:"when,omitempty"`
}
