package libyaml

type HostRequirements struct {
	ReplicatedVersion string `yaml:"replicated_version,omitempty" json:"replicated_version,omitempty" validate:"omitempty,semverrange"`
	DockerVersion     string `yaml:"docker_version,omitempty" json:"docker_version,omitempty" validate:"omitempty,dockerversion"`
	CPUCores          uint   `yaml:"cpu_cores,omitempty" json:"cpu_cores,omitempty"`
	CPUMhz            uint   `yaml:"cpu_mhz,omitempty" json:"cpu_mhz,omitempty"`
	Memory            string `yaml:"memory,omitempty" json:"memory,omitempty" validate:"omitempty,bytes"`
	DiskSpace         string `yaml:"disk_space,omitempty" json:"disk_space,omitempty" validate:"omitempty,bytes"`
	DockerSpace       string `yaml:"docker_space,omitempty" json:"docker_space,omitempty" validate:"omitempty,bytes"`
}
