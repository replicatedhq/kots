package libyaml

import "encoding/json"

var (
	AdminCommandRunTypeExec AdminCommandRunType = "exec"
)

type AdminCommand struct {
	// AdminCommandV2 api version >= 2.6.0
	AdminCommandV2 `yaml:",inline"`
	// AdminCommandV1 api version < 2.6.0
	AdminCommandV1 `yaml:",inline"`
}

type AdminCommandV2 struct {
	Alias   string                   `yaml:"alias" json:"alias" validate:"required,shellalias"`
	Command []string                 `yaml:"command,flow" json:"command" validate:"required"`
	Timeout uint                     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	RunType AdminCommandRunType      `yaml:"run_type,omitempty" json:"run_type,omitempty"` // default "exec"
	When    string                   `yaml:"when,omitempty" json:"when,omitempty"`
	Source  SchedulerContainerSource `yaml:"source" json:"source" validate:"required"`
}

type AdminCommandRunType string

type AdminCommandV1 struct { // deprecated
	Component string        `yaml:"component,omitempty" json:"component,omitempty" validate:"omitempty,componentexists"`
	Image     *CommandImage `yaml:"image,omitempty" json:"image,omitempty" validate:"omitempty,dive"`
}

type CommandImage struct {
	Name    string `yaml:"image_name" json:"image_name" validate:"required"`
	Version string `yaml:"version" json:"version"`
}

func (c *AdminCommand) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return c.unmarshal(unmarshal)
}

func (c *AdminCommand) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}
	return c.unmarshal(unmarshal)
}

func (c *AdminCommand) unmarshal(unmarshal func(interface{}) error) error {
	v2 := AdminCommandV2{}
	if err := unmarshal(&v2); err != nil {
		return err
	}
	c.AdminCommandV2 = v2

	v1 := AdminCommandV1{}
	if err := unmarshal(&v1); err != nil {
		return err
	}
	c.AdminCommandV1 = v1

	// if any are already set its probably not the "inline" style
	if c.Source.SourceContainerNative == nil && c.Source.SourceContainerSwarm == nil && c.Source.SourceContainerK8s == nil {
		if err := UnmarshalInline(unmarshal, &c.Source); err != nil {
			return err
		}
	}

	// backwards compatibility
	if c.Source.SourceContainerNative != nil {
		if c.Image == nil {
			c.Image = &CommandImage{}
		}

		if c.Component == "" {
			c.Component = c.Source.SourceContainerNative.Component
		}

		if c.Source.SourceContainerNative.Container == "" {
			c.Source.SourceContainerNative.Container = c.Image.Name
		} else {
			c.Image.Name = c.Source.SourceContainerNative.Container
		}
	}

	return nil
}
