package libyaml

type ContainerEvent struct {
	Name    string   `yaml:"name" json:"name"`
	Trigger string   `yaml:"trigger" json:"trigger"`
	Data    string   `yaml:"data" json:"data"`
	Args    []string `yaml:"args" json:"args"`
	// Timeout in seconds (0 is default 10 minutes, -1 is no timeout)
	Timeout       int                           `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Subscriptions []*ContainerEventSubscription `yaml:"subscriptions" json:"subscriptions" validate:"dive,exists"`
}
