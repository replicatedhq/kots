package libyaml

type Component struct {
	Name             string                    `yaml:"name" json:"name" validate:"required"`
	Tags             []string                  `yaml:"tags,omitempty" json:"tags,omitempty"`
	Conflicts        []string                  `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Cluster          BoolString                `yaml:"cluster" json:"cluster"`
	ClusterHostCount ComponentClusterHostCount `yaml:"cluster_host_count,omitempty" json:"cluster_host_count,omitempty"`
	HostRequirements HostRequirements          `yaml:"host_requirements,omitempty" json:"host_requirements,omitempty"`
	LogOptions       LogOptions                `yaml:"logs" json:"logs"`
	HostVolumes      []*HostVolume             `yaml:"host_volumes,omitempty" json:"host_volumes,omitempty"`
	Containers       []*Container              `yaml:"containers" json:"containers" validate:"dive,exists"` // TODO: validate:"min=1"
}

type ComponentClusterHostCount struct {
	// Strategy = "autoscale" api version >= 2.7.0
	// Strategy = "random" api version >= 2.5.0
	Strategy          string     `yaml:"strategy,omitempty" json:"strategy,omitempty" validate:"omitempty,clusterstrategy"`
	Min               UintString `yaml:"min" json:"min" validate:"omitempty,uint"`
	Max               UintString `yaml:"max,omitempty" json:"max,omitempty" validate:"omitempty,uint"` // 0 == unlimited
	ThresholdHealthy  UintString `yaml:"threshold_healthy,omitempty" json:"threshold_healthy,omitempty" validate:"omitempty,uint"`
	ThresholdDegraded UintString `yaml:"threshold_degraded,omitempty" json:"threshold_degraded,omitempty" validate:"omitempty,uint"` // 0 == no degraded state
}
