package libyaml

import "encoding/json"

type Message struct {
	ID             string                 `yaml:"id,omitempty" json:"id,omitempty"`
	DefaultMessage string                 `yaml:"default_message" json:"default_message" validate:"required"`
	Args           map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`
}

func (m *Message) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return m.unmarshal(unmarshal)
}

func (m *Message) UnmarshalJSON(data []byte) error {
	unmarshal := func(v interface{}) error {
		return json.Unmarshal(data, v)
	}
	return m.unmarshal(unmarshal)
}

func (m *Message) unmarshal(unmarshal func(interface{}) error) error {
	var outstr string
	err := unmarshal(&outstr)
	if err == nil {
		// this can be a string or a complex type message for localization
		m.DefaultMessage = outstr
		return nil
	}
	out := struct {
		ID             string                 `yaml:"id" json:"id"`
		DefaultMessage string                 `yaml:"default_message" json:"default_message"`
		Args           map[string]interface{} `yaml:"args" json:"args"`
	}{}
	err = unmarshal(&out)
	if err != nil {
		return err
	}
	m.ID = out.ID
	m.DefaultMessage = out.DefaultMessage
	m.Args = out.Args
	return nil
}
