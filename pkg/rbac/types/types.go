package types

type Group struct {
	ID      string   `json:"id" yaml:"id"`
	RoleIDs []string `json:"roleIds" yaml:"roleIds"`
}
type Role struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Allow       []Policy `json:"allow" yaml:"allow"`
	Deny        []Policy `json:"deny" yaml:"deny"`
}

type Policy struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Action      string `json:"action" yaml:"action"`
	Resource    string `json:"resource" yaml:"resource"`
}
