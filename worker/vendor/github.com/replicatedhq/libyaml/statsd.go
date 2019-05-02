package libyaml

type StatsD struct {
	Port int32 `yaml:"port" json:"port" validate:"tcpport"`
}
