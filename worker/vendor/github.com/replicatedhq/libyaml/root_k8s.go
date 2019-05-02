package libyaml

type RootK8s struct {
	PVClaims []K8sPVClaim `yaml:"volume_claims,omitempty" json:"volume_claims,omitempty"`
}
