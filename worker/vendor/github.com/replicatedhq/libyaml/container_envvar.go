package libyaml

import "encoding/json"

type ContainerEnvVar struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
	// deprecated: use Value instead
	StaticVal             string `yaml:"static_val" json:"static_val"`
	IsExcludedFromSupport string `yaml:"is_excluded_from_support" json:"is_excluded_from_support"`
	When                  string `yaml:"when" json:"when"`
}

func (c *ContainerEnvVar) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return c.unmarshal(unmarshal)
}

func (c *ContainerEnvVar) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}
	return c.unmarshal(unmarshal)
}

func (c *ContainerEnvVar) unmarshal(unmarshal func(interface{}) error) error {
	internal := struct {
		Name                  string `yaml:"name" json:"name"`
		Value                 string `yaml:"value" json:"value"`
		StaticVal             string `yaml:"static_val" json:"static_val"`
		IsExcludedFromSupport string `yaml:"is_excluded_from_support" json:"is_excluded_from_support"`
		When                  string `yaml:"when" json:"when"`
	}{}
	if err := unmarshal(&internal); err != nil {
		return err
	}
	c.Name = internal.Name
	c.IsExcludedFromSupport = internal.IsExcludedFromSupport
	c.When = internal.When
	if internal.Value != "" {
		c.Value = internal.Value
		c.StaticVal = internal.Value
	} else if internal.StaticVal != "" {
		c.Value = internal.StaticVal
		c.StaticVal = internal.StaticVal
	}
	return nil
}
