package types

type Config struct {
	Enabled  bool            `json:"enabled" yaml:"enabled"`
	Ingress  *IngressConfig  `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	NodePort *NodePortConfig `json:"nodePort,omitempty" yaml:"nodePort,omitempty"`
	External *ExternalConfig `json:"external,omitempty" yaml:"external,omitempty"`
	// TODO: Service type LoadBalancer
}

type IngressConfig struct {
	Address       string            `json:"address,omitempty" yaml:"address,omitempty"` // if address is empty it is inferred
	Path          string            `json:"path" yaml:"path"`
	Host          string            `json:"host" yaml:"host"`
	TLSSecretName string            `json:"tlsSecretName,omitempty" yaml:"tlsSecretName,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

type NodePortConfig struct {
	Address string `json:"address" yaml:"address"`
	Port    int    `json:"port" yaml:"port"`
}

type ExternalConfig struct {
	Address string `json:"address" yaml:"address"`
}
