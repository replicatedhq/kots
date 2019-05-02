package libyaml

import "strconv"

type BoolString string

func (s BoolString) Parse() (bool, error) {
	return strconv.ParseBool(string(s))
}

// TODO: json?

func (s BoolString) MarshalYAML() (interface{}, error) {
	if s == "" {
		return false, nil
	}
	b, err := s.Parse()
	if err == nil {
		return b, nil
	}
	return string(s), nil
}
