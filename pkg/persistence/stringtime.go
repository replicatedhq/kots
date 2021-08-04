package persistence

import "time"

type StringTime struct {
	Time time.Time
}

// this seems to be the format that we are using!
//  2021-08-04 16:17:10.241794246+00:00
// it's rfc3339-like, but not exactly
var (
	formatString = "2006-01-02 15:04:05.999999999-07:00"
)

// Scan implements the Scanner interface.
func (s *StringTime) Scan(value interface{}) error {
	switch v := value.(type) {
	case *time.Time:
		s.Time = *v
		break
	case string:
		t, err := time.Parse(formatString, v)
		if err != nil {
			return err
		}
		s.Time = t
		break
	}

	return nil
}

type NullStringTime struct {
	Time  time.Time
	Valid bool
}

// Scan implements the Scanner interface.
func (s *NullStringTime) Scan(value interface{}) error {
	if value == nil {
		s.Valid = false
		return nil
	}

	switch v := value.(type) {
	case *time.Time:
		s.Time = *v
		s.Valid = true
		break
	case string:
		t, err := time.Parse(formatString, v)
		if err != nil {
			return err
		}
		s.Time = t
		s.Valid = true
		break
	}

	return nil
}
