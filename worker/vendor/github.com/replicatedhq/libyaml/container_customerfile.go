package libyaml

type ContainerCustomerFile struct {
	Id        string `yaml:"name" json:"name"`
	Filename  string `yaml:"filename" json:"filename"`
	When      string `yaml:"when" json:"when"`
	FileMode  string `yaml:"file_mode" json:"file_mode"`
	FileOwner string `yaml:"file_owner" json:"file_owner"`
}
