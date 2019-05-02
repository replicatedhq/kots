package libyaml

type Terms struct {
	Markdown string `yaml:"markdown" json:"markdown"`
	Version  int    `yaml:"version,omitempty" json:"version,omitempty"`
}
