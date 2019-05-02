package libyaml

type ConfigGroup struct {
	Name        string        `yaml:"name" json:"name" validate:"required"`
	Title       string        `yaml:"title" json:"title"`
	Description string        `yaml:"description" json:"description"`
	TestProc    *TestProc     `yaml:"test_proc" json:"test_proc"`
	When        string        `yaml:"when" json:"when" validate:"configitemwhen"`
	Filters     []string      `yaml:"filters" json:"filters" validate:"dive,exists"`
	Items       []*ConfigItem `yaml:"items" json:"items" validate:"dive,exists"`
}
