package libyaml

type IdentitySource struct {
	Source  string `yaml:"source" json:"source"`
	Filter  string `yaml:"filter" json:"filter"`
	Enabled string `yaml:"enabled" json:"enabled"`
}
