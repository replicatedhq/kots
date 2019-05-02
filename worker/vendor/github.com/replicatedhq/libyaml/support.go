package libyaml

import "encoding/json"

type Support struct {
	Files    []SupportFile    `yaml:"files,omitempty" json:"files,omitempty" validate:"dive"`
	Commands []SupportCommand `yaml:"commands,omitempty" json:"commands,omitempty" validate:"dive"`
	Timeout  UintString       `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

type SupportFile struct {
	Filename string                   `yaml:"filename" json:"filename" validate:"required"`
	Source   SchedulerContainerSource `yaml:"source" json:"source" validate:"required"`
}

type SupportCommand struct {
	Filename string                   `yaml:"filename" json:"filename" validate:"required"`
	Command  []string                 `yaml:"command" json:"command" validate:"required"`
	Source   SchedulerContainerSource `yaml:"source" json:"source" validate:"required"`
}

func (s *SupportFile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return s.unmarshal(unmarshal)
}

func (s *SupportFile) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}
	return s.unmarshal(unmarshal)
}

func (s *SupportFile) unmarshal(unmarshal func(interface{}) error) error {
	internal := struct {
		Filename string                   `yaml:"filename" json:"filename"`
		Source   SchedulerContainerSource `yaml:"source" json:"source"`
	}{}
	if err := unmarshal(&internal); err != nil {
		return err
	}
	s.Filename = internal.Filename
	s.Source = internal.Source
	// if any are already set its probably not the "inline" style
	if s.Source.SourceContainerNative != nil || s.Source.SourceContainerSwarm != nil || s.Source.SourceContainerK8s != nil {
		return nil
	}
	return UnmarshalInline(unmarshal, &s.Source)
}

func (s *SupportCommand) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return s.unmarshal(unmarshal)
}

func (s *SupportCommand) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}
	return s.unmarshal(unmarshal)
}

func (s *SupportCommand) unmarshal(unmarshal func(interface{}) error) error {
	internal := struct {
		Filename string                   `yaml:"filename" json:"filename"`
		Command  []string                 `yaml:"command" json:"command"`
		Source   SchedulerContainerSource `yaml:"source" json:"source"`
	}{}
	if err := unmarshal(&internal); err != nil {
		return err
	}
	s.Filename = internal.Filename
	s.Command = internal.Command
	s.Source = internal.Source
	// if any are already set its probably not the "inline" style
	if s.Source.SourceContainerNative != nil || s.Source.SourceContainerSwarm != nil || s.Source.SourceContainerK8s != nil {
		return nil
	}
	return UnmarshalInline(unmarshal, &s.Source)
}
