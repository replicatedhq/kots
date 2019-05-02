package libyaml

type Image struct {
	Source       string        `yaml:"source,omitempty" json:"source,omitempty" validate:"externalregistryexists"` // default public
	Name         string        `yaml:"name" json:"name" validate:"required"`
	Tag          string        `yaml:"tag,omitempty" json:"tag,omitempty"`                          // default latest
	Key          string        `yaml:"key,omitempty" json:"key,omitempty" validate:"isempty"`       // image key that identifies linked registry
	Domain       string        `yaml:"domain,omitempty" json:"domain,omitempty" validate:"isempty"` // linked registry domain
	ContentTrust *ContentTrust `yaml:"content_trust,omitempty" json:"content_trust,omitempty"`
}
