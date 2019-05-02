package libyaml

type Localization struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	LocalesEnabled []string `yaml:"locales_enabled,omitempty" json:"locales_enabled,omitempty" validate:"omitempty,min=1"`
	Locales        []Locale `yaml:"locales,omitempty" json:"locales,omitempty" validate:"dive"`
}

type Locale struct {
	Tag          string            `yaml:"tag" json:"tag" validate:"required"`
	Translations map[string]string `yaml:"translations" json:"translations"`
}
