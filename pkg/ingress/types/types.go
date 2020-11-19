package types

import (
	extensions "k8s.io/api/extensions/v1beta1"
)

type Config struct {
	Annotations map[string]string       `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Path        string                  `json:"path" yaml:"path"`
	Host        string                  `json:"host" yaml:"host"`
	TLS         []extensions.IngressTLS `json:"tls,omitempty" yaml:"tls,omitempty"`
}

func (c Config) GetPath(dflt string) string {
	if c.Path != "" {
		return c.Path
	}

	if c.Host != "" {
		return ""
	}

	return dflt
}
