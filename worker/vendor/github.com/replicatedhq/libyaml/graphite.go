package libyaml

type Graphite struct {
	Port int32 `yaml:"port" json:"port" validate:"tcpport"`
}
