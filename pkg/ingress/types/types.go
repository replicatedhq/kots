package types

import (
	extensions "k8s.io/api/extensions/v1beta1"
)

type Config struct {
	Kotsadm Kotsadm `json:"kotsadm" yaml:"kotsadm"`
	Dex     Dex     `json:"dex,omitempty" yaml:"dex,omitempty"`
}

type Kotsadm struct {
	Annotations map[string]string       `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Path        string                  `json:"path" yaml:"path"`
	Host        string                  `json:"host" yaml:"host"`
	TLS         []extensions.IngressTLS `json:"tls,omitempty" yaml:"tls,omitempty"`
}

type Dex struct {
	Annotations map[string]string       `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Path        string                  `json:"path" yaml:"path"`
	Host        string                  `json:"host" yaml:"host"`
	TLS         []extensions.IngressTLS `json:"tls,omitempty" yaml:"tls,omitempty"`
}

func (c Config) KotsadmPath() string {
	if c.Kotsadm.Path != "" {
		return c.Kotsadm.Path
	}

	if c.Kotsadm.Host != "" {
		return ""
	}

	return "/kotsadm"
}

func (c Config) DexPath() string {
	if c.Dex.Path != "" {
		return c.Dex.Path
	}

	if c.Dex.Host != "" {
		return ""
	}

	return "/dex"
}
