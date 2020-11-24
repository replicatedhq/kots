package types

type Role struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	PolicyIDs   []string `json:"policyIds" yaml:"policyIds"`
}

type Policy struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Allowed     []string `json:"allowed,omitempty" yaml:"allowed,omitempty"`
	Denied      []string `json:"denied,omitempty" yaml:"denied,omitempty"`
}

type Group struct {
	ID      string   `json:"id" yaml:"id"`
	RoleIDs []string `json:"roleIds" yaml:"roleIds"`
}
