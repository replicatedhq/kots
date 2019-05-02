package libyaml

type ContainerConfigFile struct {
	Filename  string `yaml:"filename" json:"filename" validate:"required"`
	Contents  string `yaml:"contents" json:"contents"`
	Source    string `yaml:"source" json:"source" validate:"integrationexists"`
	Owner     string `yaml:"owner" json:"owner"`
	Repo      string `yaml:"repo" json:"repo"`
	Path      string `yaml:"path" json:"path"`
	Ref       string `yaml:"ref" json:"ref"`
	FileMode  string `yaml:"file_mode" json:"file_mode"`
	FileOwner string `yaml:"file_owner" json:"file_owner"`
}
