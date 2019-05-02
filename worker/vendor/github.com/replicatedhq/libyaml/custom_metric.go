package libyaml

type CustomMetric struct {
	Target            string  `yaml:"target" json:"target" validate:"required"`
	Retention         string  `yaml:"retention" json:"retention" validate:"graphiteretention"`
	AggregationMethod string  `yaml:"aggregation_method" json:"aggregation_method" validate:"graphiteaggregation"`
	XFilesFactor      float64 `yaml:"xfiles_factor" json:"xfiles_factor"`
	Reported          string  `yaml:"reported" json:"reported"`
}
