package libyaml

type ContainerEventSubscription struct {
	ComponentName string `yaml:"component" json:"component" validate:"componentexists"`
	ContainerName string `yaml:"container" json:"container" validate:"containerexists=ComponentName"`
	Action        string `yaml:"action" json:"action"`
}
