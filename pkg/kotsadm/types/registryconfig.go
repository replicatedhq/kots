package types

type RegistryConfig struct {
	OverrideVersion   string
	OverrideRegistry  string
	OverrideNamespace string
	Username          string
	Password          string
	IsReadOnly        bool
}
