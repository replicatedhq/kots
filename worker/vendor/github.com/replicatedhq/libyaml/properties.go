package libyaml

type Properties struct {
	AppUrl              interface{} `yaml:"app_url" json:"app_url"`
	LogoUrl             string      `yaml:"logo_url" json:"logo_url" validate:"url"`
	ConsoleTitle        string      `yaml:"console_title" json:"console_title"`
	BypassLocalRegistry bool        `yaml:"bypass_local_registry" json:"bypass_local_registry"`
	ShellAlias          string      `yaml:"shell_alias" json:"shell_alias" validate:"shellalias"`
}
