package libyaml

type CustomMonitor struct {
	Name      string   `yaml:"name" json:"name" validate:"required"`
	Target    string   `yaml:"target" json:"target"` // for backwards compatibility
	Targets   []string `yaml:"targets" json:"targets"`
	Dashboard string   `yaml:"dashboard" json:"dashboard"`
	From      string   `yaml:"from" json:"from"`
	Until     string   `yaml:"until" json:"until"`

	Display struct {
		LabelUnit          string  `yaml:"label_unit" json:"label_unit"`
		LabelScale         string  `yaml:"label_scale" json:"label_scale" validate:"monitorlabelscale"`
		LabelRangeOverride bool    `yaml:"label_range_override" json:"label_range_override"`
		LabelMin           float64 `yaml:"label_min" json:"label_min"`
		LabelMax           float64 `yaml:"label_max" json:"label_max"`
		LabelCount         int     `yaml:"label_count" json:"label_count"`
		FillColor          string  `yaml:"fill_color" json:"fill_color" validate:"hexcolor|rgb|rgba|isempty"`
		StrokeColor        string  `yaml:"stroke_color" json:"stroke_color" validate:"hexcolor|rgb|rgba|isempty"`
		CssClassName       string  `yaml:"css_class_name" json:"css_class_name"`
	} `yaml:"display" json:"display" validate:"dive"`
}

type Monitors struct {
	Cpuacct []string `yaml:"cpuacct" validate:"dive,componentcontainer,componentexists,containerexists"`
	Memory  []string `yaml:"memory" validate:"dive,componentcontainer,componentexists,containerexists"`

	Custom []CustomMonitor `yaml:"custom" validate:"hastarget,dive"`
}
