package libyaml

type ContentTrust struct {
	PublicKeyFingerprint string `yaml:"public_key_fingerprint" json:"public_key_fingerprint" validate:"fingerprint"`
}
