package libyaml

type Container struct {
	Source               string                        `yaml:"source" json:"source" validate:"required,externalregistryexists"`
	ImageName            string                        `yaml:"image_name" json:"image_name" validate:"required"`
	Version              string                        `yaml:"version" json:"version" validate:"required"`
	ImageKey             string                        `yaml:"image_key,omitempty" json:"image_key,omitempty" validate:"isempty"`
	ImageDomain          string                        `yaml:"image_domain,omitempty" json:"image_domain,omitempty" validate:"isempty"`
	DisplayName          string                        `yaml:"display_name" json:"display_name"`
	Name                 string                        `yaml:"name" json:"name" validate:"containernameunique,clusterinstancefalse"`
	Privileged           bool                          `yaml:"privileged" json:"privileged"`
	NetworkMode          string                        `yaml:"network_mode" json:"network_mode"`
	CPUShares            string                        `yaml:"cpu_shares" json:"cpu_shares"`
	MemoryLimit          string                        `yaml:"memory_limit" json:"memory_limit"`
	MemorySwapLimit      string                        `yaml:"memory_swap_limit" json:"memory_swap_limit"`
	ULimits              []ULimit                      `yaml:"ulimits,omitempty" json:"ulimits,omitempty"`
	AllocateTTY          string                        `yaml:"allocate_tty" json:"allocate_tty"`
	SecurityCapAdd       []string                      `yaml:"security_cap_add" json:"security_cap_add"`
	SecurityOptions      []string                      `yaml:"security_options" json:"security_options"`
	Hostname             string                        `yaml:"hostname" json:"hostname"`
	Cmd                  string                        `yaml:"cmd" json:"cmd"`
	Entrypoint           *[]string                     `yaml:"entrypoint" json:"entrypoint"`
	Ephemeral            bool                          `yaml:"ephemeral" json:"ephemeral"`
	SuppressRestart      []string                      `yaml:"suppress_restart" json:"suppress_restart"`
	Cluster              BoolString                    `yaml:"cluster" json:"cluster" validate:"omitempty,bool"`
	Restart              *ContainerRestartPolicy       `yaml:"restart" json:"restart"`
	ClusterInstanceCount ContainerClusterInstanceCount `yaml:"cluster_instance_count" json:"cluster_instance_count"`
	PublishEvents        []*ContainerEvent             `yaml:"publish_events" json:"publish_events" validate:"dive,exists"`
	SubscribedEvents     []map[string]interface{}      `yaml:"-" json:"-"`
	ConfigFiles          []*ContainerConfigFile        `yaml:"config_files" json:"config_files" validate:"dive,exists"`
	CustomerFiles        []*ContainerCustomerFile      `yaml:"customer_files" json:"customer_files" validate:"dive,exists"`
	EnvVars              []*ContainerEnvVar            `yaml:"env_vars" json:"env_vars" validate:"dive,exists"`
	Ports                []*ContainerPort              `yaml:"ports,omitempty" json:"ports,omitempty" validate:"dive,exists"`
	LogOptions           LogOptions                    `yaml:"logs" json:"logs"`
	Volumes              []*ContainerVolume            `yaml:"volumes" json:"volumes" validate:"dive,exists"`
	VolumesFrom          []string                      `yaml:"volumes_from" json:"volumes_from" validate:"dive,required,containernameexists,requiressubscription"`
	ExtraHosts           []*ContainerExtraHost         `yaml:"extra_hosts" json:"extra_hosts" validate:"dive,exists"`
	SupportFiles         []*ContainerSupportFile       `yaml:"support_files" json:"support_files" validate:"dive,exists"`
	SupportCommands      []*ContainerSupportCommand    `yaml:"support_commands" json:"support_commands" validate:"dive,exists"`
	ContentTrust         ContentTrust                  `yaml:"content_trust" json:"content_trust"`
	When                 string                        `yaml:"when" json:"when"`
	Dynamic              string                        `yaml:"dynamic" json:"dynamic"`
	PidMode              string                        `yaml:"pid_mode" json:"pid_mode"`
	ShmSize              int64                         `yaml:"shm_size" json:"shm_size"`
	Labels               []string                      `yaml:"labels" json:"labels"`
	StopTimeout          UintString                    `yaml:"stop_timeout,omitempty" json:"stop_timeout,omitempty" validate:"omitempty,uint"`
}

type ContainerRestartPolicy struct {
	Policy string `yaml:"policy" json:"policy"`
	Max    uint   `yaml:"max" json:"max"`
}

type ContainerClusterInstanceCount struct {
	Initial           UintString `yaml:"initial" json:"initial" validate:"omitempty,uint"`
	Max               UintString `yaml:"max,omitempty" json:"max" validate:"omitempty,uint"` // 0 == unlimited
	ThresholdHealthy  UintString `yaml:"threshold_healthy,omitempty" json:"threshold_healthy,omitempty" validate:"omitempty,uint"`
	ThresholdDegraded UintString `yaml:"threshold_degraded,omitempty" json:"threshold_degraded" validate:"omitempty,uint"` // 0 == no degraded state
}

type ULimit struct {
	Name string `yaml:"name" json:"name" validate:"required"`
	Soft string `yaml:"soft,omitempty" json:"soft,omitempty"`
	Hard string `yaml:"hard,omitempty" json:"hard,omitempty"`
}

func (c *Container) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m marshallerContainer
	if err := unmarshal(&m); err != nil {
		return err
	}
	m.decode(c)

	cluster, err := c.Cluster.Parse()
	if err != nil {
		cluster = c.Cluster != "" // assume this is a template
	}
	if cluster {
		if c.ClusterInstanceCount.Initial == "" || c.ClusterInstanceCount.Initial == "0" {
			c.ClusterInstanceCount.Initial = "1"
		}
	}

	return nil
}

func (c Container) MarshalYAML() (interface{}, error) {
	cluster, err := c.Cluster.Parse()
	if err != nil {
		cluster = c.Cluster != "" // assume this is a template
	}
	if !cluster {
		m := nonclusterableContainer{}
		m.encode(c)
		return m, nil
	}

	m := marshallerContainer{}
	m.encode(c)

	if m.ClusterInstanceCount.Initial == "" || m.ClusterInstanceCount.Initial == "0" {
		m.ClusterInstanceCount.Initial = "1"
	}
	return m, nil
}

type marshallerContainer Container

func (m *marshallerContainer) encode(c Container) {
	// TODO: In go 1.8, this can be just copied automatically
	m.Source = c.Source
	m.ImageName = c.ImageName
	m.Version = c.Version
	m.ImageKey = c.ImageKey
	m.ImageDomain = c.ImageDomain
	m.DisplayName = c.DisplayName
	m.Name = c.Name
	m.Privileged = c.Privileged
	m.NetworkMode = c.NetworkMode
	m.CPUShares = c.CPUShares
	m.MemoryLimit = c.MemoryLimit
	m.MemorySwapLimit = c.MemorySwapLimit
	m.ULimits = c.ULimits
	m.AllocateTTY = c.AllocateTTY
	m.SecurityCapAdd = c.SecurityCapAdd
	m.SecurityOptions = c.SecurityOptions
	m.Hostname = c.Hostname
	m.Cmd = c.Cmd
	m.Entrypoint = c.Entrypoint
	m.Ephemeral = c.Ephemeral
	m.SuppressRestart = c.SuppressRestart
	m.Cluster = c.Cluster
	m.Restart = c.Restart
	m.ClusterInstanceCount = c.ClusterInstanceCount
	m.PublishEvents = c.PublishEvents
	m.SubscribedEvents = c.SubscribedEvents
	m.ConfigFiles = c.ConfigFiles
	m.CustomerFiles = c.CustomerFiles
	m.EnvVars = c.EnvVars
	m.Ports = c.Ports
	m.LogOptions = c.LogOptions
	m.Volumes = c.Volumes
	m.VolumesFrom = c.VolumesFrom
	m.ExtraHosts = c.ExtraHosts
	m.SupportFiles = c.SupportFiles
	m.SupportCommands = c.SupportCommands
	m.ContentTrust = c.ContentTrust
	m.When = c.When
	m.Dynamic = c.Dynamic
	m.PidMode = c.PidMode
	m.ShmSize = c.ShmSize
	m.Labels = c.Labels
	m.StopTimeout = c.StopTimeout
}

func (m marshallerContainer) decode(c *Container) {
	// TODO: In go 1.8, this can be just copied automatically
	c.Source = m.Source
	c.ImageName = m.ImageName
	c.Version = m.Version
	c.ImageKey = m.ImageKey
	c.ImageDomain = m.ImageDomain
	c.DisplayName = m.DisplayName
	c.Name = m.Name
	c.Privileged = m.Privileged
	c.NetworkMode = m.NetworkMode
	c.CPUShares = m.CPUShares
	c.MemoryLimit = m.MemoryLimit
	c.MemorySwapLimit = m.MemorySwapLimit
	c.ULimits = m.ULimits
	c.AllocateTTY = m.AllocateTTY
	c.SecurityCapAdd = m.SecurityCapAdd
	c.SecurityOptions = m.SecurityOptions
	c.Hostname = m.Hostname
	c.Cmd = m.Cmd
	c.Entrypoint = m.Entrypoint
	c.Ephemeral = m.Ephemeral
	c.SuppressRestart = m.SuppressRestart
	c.Cluster = m.Cluster
	c.Restart = m.Restart
	c.ClusterInstanceCount = m.ClusterInstanceCount
	c.PublishEvents = m.PublishEvents
	c.SubscribedEvents = m.SubscribedEvents
	c.ConfigFiles = m.ConfigFiles
	c.CustomerFiles = m.CustomerFiles
	c.EnvVars = m.EnvVars
	c.Ports = m.Ports
	c.LogOptions = m.LogOptions
	c.Volumes = m.Volumes
	c.VolumesFrom = m.VolumesFrom
	c.ExtraHosts = m.ExtraHosts
	c.SupportFiles = m.SupportFiles
	c.SupportCommands = m.SupportCommands
	c.ContentTrust = m.ContentTrust
	c.When = m.When
	c.Dynamic = m.Dynamic
	c.PidMode = m.PidMode
	c.ShmSize = m.ShmSize
	c.Labels = m.Labels
	c.StopTimeout = m.StopTimeout
}

type nonclusterableContainer struct {
	Source           string                     `yaml:"source" json:"source" validate:"required,externalregistryexists"`
	ImageName        string                     `yaml:"image_name" json:"image_name" validate:"required"`
	Version          string                     `yaml:"version" json:"version" validate:"required"`
	ImageKey         string                     `yaml:"image_key,omitempty" json:"image_key,omitempty" validate:"isempty"`
	ImageDomain      string                     `yaml:"image_domain,omitempty" json:"image_domain,omitempty" validate:"isempty"`
	DisplayName      string                     `yaml:"display_name" json:"display_name"`
	Name             string                     `yaml:"name" json:"name" validate:"containernameunique,clusterinstancefalse"`
	Privileged       bool                       `yaml:"privileged" json:"privileged"`
	NetworkMode      string                     `yaml:"network_mode" json:"network_mode"`
	CPUShares        string                     `yaml:"cpu_shares" json:"cpu_shares"`
	MemoryLimit      string                     `yaml:"memory_limit" json:"memory_limit"`
	MemorySwapLimit  string                     `yaml:"memory_swap_limit" json:"memory_swap_limit"`
	ULimits          []ULimit                   `yaml:"ulimits" json:"ulimits"`
	AllocateTTY      string                     `yaml:"allocate_tty" json:"allocate_tty"`
	SecurityCapAdd   []string                   `yaml:"security_cap_add" json:"security_cap_add"`
	SecurityOptions  []string                   `yaml:"security_options" json:"security_options"`
	Hostname         string                     `yaml:"hostname" json:"hostname"`
	Cmd              string                     `yaml:"cmd" json:"cmd"`
	Entrypoint       *[]string                  `yaml:"entrypoint" json:"entrypoint"`
	Ephemeral        bool                       `yaml:"ephemeral" json:"ephemeral"`
	SuppressRestart  []string                   `yaml:"suppress_restart" json:"suppress_restart"`
	Cluster          BoolString                 `yaml:"cluster" json:"cluster" validate:"omitempty,bool"`
	Restart          *ContainerRestartPolicy    `yaml:"restart" json:"restart"`
	PublishEvents    []*ContainerEvent          `yaml:"publish_events" json:"publish_events" validate:"dive,exists"`
	SubscribedEvents []map[string]interface{}   `yaml:"-" json:"-"`
	ConfigFiles      []*ContainerConfigFile     `yaml:"config_files" json:"config_files" validate:"dive,exists"`
	CustomerFiles    []*ContainerCustomerFile   `yaml:"customer_files" json:"customer_files" validate:"dive,exists"`
	EnvVars          []*ContainerEnvVar         `yaml:"env_vars" json:"env_vars" validate:"dive,exists"`
	Ports            []*ContainerPort           `yaml:"ports,omitempty" json:"ports,omitempty" validate:"dive,exists"`
	LogOptions       LogOptions                 `yaml:"logs" json:"logs"`
	Volumes          []*ContainerVolume         `yaml:"volumes" json:"volumes" validate:"dive,exists"`
	VolumesFrom      []string                   `yaml:"volumes_from" json:"volumes_from" validate:"dive,required,containernameexists"`
	ExtraHosts       []*ContainerExtraHost      `yaml:"extra_hosts" json:"extra_hosts" validate:"dive,exists"`
	SupportFiles     []*ContainerSupportFile    `yaml:"support_files" json:"support_files" validate:"dive,exists"`
	SupportCommands  []*ContainerSupportCommand `yaml:"support_commands" json:"support_commands" validate:"dive,exists"`
	ContentTrust     ContentTrust               `yaml:"content_trust" json:"content_trust" validate:"dive"`
	When             string                     `yaml:"when" json:"when"`
	Dynamic          string                     `yaml:"dynamic" json:"dynamic"`
	PidMode          string                     `yaml:"pid_mode" json:"pid_mode"`
	ShmSize          int64                      `yaml:"shm_size" json:"shm_size"`
	Labels           []string                   `yaml:"labels" json:"labels"`
	StopTimeout      UintString                 `yaml:"stop_timeout,omitempty" json:"stop_timeout,omitempty" validate:"omitempty,uint"`
}

func (m *nonclusterableContainer) encode(c Container) {
	// TODO: In go 1.8, this can be just copied automatically
	m.Source = c.Source
	m.ImageName = c.ImageName
	m.Version = c.Version
	m.ImageKey = c.ImageKey
	m.ImageDomain = c.ImageDomain
	m.DisplayName = c.DisplayName
	m.Name = c.Name
	m.Privileged = c.Privileged
	m.NetworkMode = c.NetworkMode
	m.CPUShares = c.CPUShares
	m.MemoryLimit = c.MemoryLimit
	m.MemorySwapLimit = c.MemorySwapLimit
	m.ULimits = c.ULimits
	m.AllocateTTY = c.AllocateTTY
	m.SecurityCapAdd = c.SecurityCapAdd
	m.SecurityOptions = c.SecurityOptions
	m.Hostname = c.Hostname
	m.Cmd = c.Cmd
	m.Entrypoint = c.Entrypoint
	m.Ephemeral = c.Ephemeral
	m.SuppressRestart = c.SuppressRestart
	m.Cluster = "false"
	m.Restart = c.Restart
	m.PublishEvents = c.PublishEvents
	m.SubscribedEvents = c.SubscribedEvents
	m.ConfigFiles = c.ConfigFiles
	m.CustomerFiles = c.CustomerFiles
	m.EnvVars = c.EnvVars
	m.Ports = c.Ports
	m.LogOptions = c.LogOptions
	m.Volumes = c.Volumes
	m.VolumesFrom = c.VolumesFrom
	m.ExtraHosts = c.ExtraHosts
	m.SupportFiles = c.SupportFiles
	m.SupportCommands = c.SupportCommands
	m.ContentTrust = c.ContentTrust
	m.When = c.When
	m.Dynamic = c.Dynamic
	m.PidMode = c.PidMode
	m.ShmSize = c.ShmSize
	m.Labels = c.Labels
	m.StopTimeout = c.StopTimeout
}
