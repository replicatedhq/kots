package libyaml

import "encoding/json"

type SchedulerContainerSource struct {
	SourceContainerNative *SourceContainerNative `yaml:"replicated,omitempty" json:"replicated,omitempty" validate:"omitempty,dive"`
	SourceContainerSwarm  *SourceContainerSwarm  `yaml:"swarm,omitempty" json:"swarm,omitempty" validate:"omitempty,dive"`
	SourceContainerK8s    *SourceContainerK8s    `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty" validate:"omitempty,dive"`
}

type SourceContainerNative struct {
	Component string `yaml:"component" json:"component" validate:"required,componentexists"`
	Container string `yaml:"container" json:"container" validate:"containerexists=Component"`
}

type SourceContainerSwarm struct {
	Service string `yaml:"service" json:"service" validate:"required"`
}

type SourceContainerK8s struct {
	Selector  map[string]string `yaml:"selector" json:"selector" validate:"required,dive,required"`
	Selectors map[string]string `yaml:"selectors" json:"selectors"` // deprecated
	Container string            `yaml:"container,omitempty" json:"container,omitempty"`
}

func (s *SchedulerContainerSource) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return s.unmarshal(unmarshal)
}

func (s *SchedulerContainerSource) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}
	return s.unmarshal(unmarshal)
}

// UnmarshalInline can be called inside of a parent's unmarshal function to allow
func UnmarshalInline(unmarshal func(interface{}) error, s *SchedulerContainerSource) error {
	var native SourceContainerNative
	if err := unmarshal(&native); err != nil {
		return err
	}
	if native.Component != "" {
		s.SourceContainerNative = &native
		return nil
	}
	var swarm SourceContainerSwarm
	if err := unmarshal(&swarm); err != nil {
		return err
	}
	if swarm.Service != "" {
		s.SourceContainerSwarm = &swarm
		return nil
	}
	var k8s SourceContainerK8s
	if err := unmarshal(&k8s); err != nil {
		return err
	}
	// container is kinda ambiguous, should determine if selector is required
	if k8s.Selector != nil || k8s.Container != "" {
		s.SourceContainerK8s = &k8s
		if len(s.SourceContainerK8s.Selector) > 0 {
			s.SourceContainerK8s.Selectors = s.SourceContainerK8s.Selector
		} else if len(s.SourceContainerK8s.Selectors) > 0 {
			s.SourceContainerK8s.Selector = s.SourceContainerK8s.Selectors
		}
		return nil
	}
	return nil
}

type schedulerContainerSourceInternal struct {
	*SourceContainerNative `yaml:"replicated" json:"replicated"`
	*SourceContainerSwarm  `yaml:"swarm" json:"swarm"`
	*SourceContainerK8s    `yaml:"kubernetes" json:"kubernetes"`
}

func (s *SchedulerContainerSource) unmarshal(unmarshal func(interface{}) error) error {
	var internal schedulerContainerSourceInternal
	if err := unmarshal(&internal); err != nil {
		return err
	}
	if internal.SourceContainerNative != nil || internal.SourceContainerSwarm != nil || internal.SourceContainerK8s != nil {
		s.SourceContainerNative = internal.SourceContainerNative
		s.SourceContainerSwarm = internal.SourceContainerSwarm
		s.SourceContainerK8s = internal.SourceContainerK8s
		if s.SourceContainerK8s != nil {
			if len(s.SourceContainerK8s.Selector) > 0 {
				s.SourceContainerK8s.Selectors = s.SourceContainerK8s.Selector
			} else if len(s.SourceContainerK8s.Selectors) > 0 {
				s.SourceContainerK8s.Selector = s.SourceContainerK8s.Selectors
			}
		}
		return nil
	}
	return UnmarshalInline(unmarshal, s)
}

func (s SourceContainerK8s) MarshalYAML() (interface{}, error) {
	return s.marshal()
}

func (s SourceContainerK8s) MarshalJSON() ([]byte, error) {
	out, _ := s.marshal()
	return json.Marshal(out)
}

func (s SourceContainerK8s) marshal() (interface{}, error) {
	if len(s.Selector) > 0 {
		s.Selectors = s.Selector
	} else if len(s.Selectors) > 0 {
		s.Selector = s.Selectors
	}
	return s, nil
}
