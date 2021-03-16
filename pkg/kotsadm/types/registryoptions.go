package types

type KotsadmOptions struct {
	OverrideVersion   string
	OverrideRegistry  string
	OverrideNamespace string
	Username          string
	Password          string
	IsReadOnly        bool
}
