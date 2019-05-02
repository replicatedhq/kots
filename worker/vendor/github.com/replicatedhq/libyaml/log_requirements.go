package libyaml

type LogOptions struct {
	MaxSize  string `yaml:"max_size" json:"max_size"`
	MaxFiles string `yaml:"max_files" json:"max_files"`
}
