package libyaml

var (
	_ APIVersioner = &RootConfig{} // to make sure we implement the interface
)

type APIVersioner interface {
	GetAPIVersion() string
}

type RootConfig struct {
	APIVersion             string           `yaml:"replicated_api_version" json:"replicated_api_version" validate:"required,apiversion"`
	Name                   string           `yaml:"name" json:"name"`
	Version                string           `yaml:"version" json:"version"`
	ReleaseNotes           string           `yaml:"release_notes" json:"release_notes"`
	ConsoleSupportMarkdown string           `yaml:"console_support_markdown" json:"console_support_markdown"`
	Terms                  *Terms           `yaml:"terms,omitempty" json:"terms,omitempty"` // api version >= 2.27.0
	Properties             Properties       `yaml:"properties" json:"properties"`
	Identity               Identity         `yaml:"identity" json:"identity"`
	State                  State            `yaml:"state" json:"state"`
	Backup                 Backup           `yaml:"backup" json:"backup"`
	Monitors               Monitors         `yaml:"monitors" json:"monitors"`
	HostRequirements       HostRequirements `yaml:"host_requirements" json:"host_requirements"`
	// CustomRequirements api version >= 2.4.0
	CustomRequirements []CustomRequirement `yaml:"custom_requirements,omitempty" json:"custom_requirements,omitempty" validate:"dive"`
	ConfigCommands     []*ConfigCommand    `yaml:"cmds" json:"cmds" validate:"dive,exists"`
	ConfigGroups       []*ConfigGroup      `yaml:"config" json:"config" validate:"dive,exists"`
	AdminCommands      []*AdminCommand     `yaml:"admin_commands" json:"admin_commands" validate:"dive,exists"`
	CustomMetrics      []*CustomMetric     `yaml:"custom_metrics" json:"custom_metrics" validate:"dive"`
	Graphite           Graphite            `yaml:"graphite" json:"graphite" validate:"dive"`
	StatsD             StatsD              `yaml:"statsd" json:"statsd" validate:"dive"`
	Localization       *Localization       `yaml:"localization,omitempty" json:"localization,omitempty" validate:"omitempty,dive"`

	// Support api version >= 2.10.0
	Support *Support `yaml:"support,omitempty" json:"support,omitempty" validate:"omitempty,dive"`

	// Images api version >= 2.8.0
	Images []Image `yaml:"images,omitempty" json:"images,omitempty" validate:"dive"`

	Components []*Component `yaml:"components,omitempty" json:"components,omitempty" validate:"dive,exists"` // replicated scheduler config
	K8s        *K8s         `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty"`

	// Swarm api version >= 2.7.0
	Swarm *Swarm `yaml:"swarm,omitempty" json:"swarm,omitempty"`
}

func (r *RootConfig) GetAPIVersion() string {
	return r.APIVersion
}
