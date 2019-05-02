package api

import "github.com/replicatedhq/libyaml"

// Config is the top level config object
type Config struct {
	V1 []libyaml.ConfigGroup `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}
