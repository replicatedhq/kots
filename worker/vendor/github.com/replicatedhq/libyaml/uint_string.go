package libyaml

import "strconv"

type UintString string

func (s UintString) Parse() (uint64, error) {
	return strconv.ParseUint(string(s), 10, 64)
}

// TODO: json?

func (s UintString) MarshalYAML() (interface{}, error) {
	if s == "" {
		return 0, nil
	}
	i, err := s.Parse()
	if err == nil {
		return i, nil
	}
	return string(s), nil
}
