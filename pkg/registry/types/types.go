package types

type RegistrySettings struct {
	Hostname   string
	Username   string
	Password   string
	Namespace  string
	IsReadOnly bool
}

const (
	PasswordMask = "***HIDDEN***"
)

func (s RegistrySettings) IsValid() bool {
	return s.Hostname != ""
}
