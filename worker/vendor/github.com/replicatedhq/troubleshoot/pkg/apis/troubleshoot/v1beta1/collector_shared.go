package v1beta1

type CollectorMeta struct {
	CollectorName string `json:"collectorName,omitempty" yaml:"collectorName,omitempty"`
}

type ClusterInfo struct {
}

type ClusterResources struct {
}

type Secret struct {
	CollectorMeta `json:",inline" yaml:",inline"`
	Name          string `json:"name" yaml:"name"`
	Namespace     string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Key           string `json:"key,omitempty" yaml:"key,omitempty"`
	IncludeValue  bool   `json:"includeValue,omitempty" yaml:"includeValue,omitempty"`
}

type LogLimits struct {
	MaxAge   string `json:"maxAge,omitempty" yaml:"maxAge,omitempty"`
	MaxLines int64  `json:"maxLines,omitempty" yaml:"maxLines,omitempty"`
}

type Logs struct {
	CollectorMeta `json:",inline" yaml:",inline"`
	Selector      []string   `json:"selector" yaml:"selector"`
	Namespace     string     `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Limits        *LogLimits `json:"limits,omitempty" yaml:"omitempty"`
}

type Run struct {
	CollectorMeta   `json:",inline" yaml:",inline"`
	Namespace       string   `json:"namespace" yaml:"namespace"`
	Image           string   `json:"image" yaml:"image"`
	Command         []string `json:"command,omitempty" yaml:"command,omitempty"`
	Args            []string `json:"args,omitempty" yaml:"args,omitempty"`
	Timeout         string   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	ImagePullPolicy string   `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`
}

type Exec struct {
	CollectorMeta `json:",inline" yaml:",inline"`
	Selector      []string `json:"selector" yaml:"selector"`
	Namespace     string   `json:"namespace" yaml:"namespace"`
	ContainerName string   `json:"containerName,omitempty" yaml:"containerName,omitempty"`
	Command       []string `json:"command,omitempty" yaml:"command,omitempty"`
	Args          []string `json:"args,omitempty" yaml:"args,omitempty"`
	Timeout       string   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

type Copy struct {
	CollectorMeta `json:",inline" yaml:",inline"`
	Selector      []string `json:"selector" yaml:"selector"`
	Namespace     string   `json:"namespace" yaml:"namespace"`
	ContainerPath string   `json:"containerPath" yaml:"containerPath"`
	ContainerName string   `json:"containerName,omitempty" yaml:"containerName,omitempty"`
}

type HTTP struct {
	CollectorMeta `json:",inline" yaml:",inline"`
	Get           *Get  `json:"get,omitempty" yaml:"get,omitempty"`
	Post          *Post `json:"post,omitempty" yaml:"post,omitempty"`
	Put           *Put  `json:"put,omitempty" yaml:"put,omitempty"`
}

type Get struct {
	URL                string            `json:"url" yaml:"url"`
	InsecureSkipVerify bool              `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
	Headers            map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

type Post struct {
	URL                string            `json:"url" yaml:"url"`
	InsecureSkipVerify bool              `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
	Headers            map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body               string            `json:"body,omitempty" yaml:"body,omitempty"`
}

type Put struct {
	URL                string            `json:"url" yaml:"url"`
	InsecureSkipVerify bool              `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
	Headers            map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body               string            `json:"body,omitempty" yaml:"body,omitempty"`
}

type Collect struct {
	ClusterInfo      *ClusterInfo      `json:"clusterInfo,omitempty" yaml:"clusterInfo,omitempty"`
	ClusterResources *ClusterResources `json:"clusterResources,omitempty" yaml:"clusterResources,omitempty"`
	Secret           *Secret           `json:"secret,omitempty" yaml:"secret,omitempty"`
	Logs             *Logs             `json:"logs,omitempty" yaml:"logs,omitempty"`
	Run              *Run              `json:"run,omitempty" yaml:"run,omitempty"`
	Exec             *Exec             `json:"exec,omitempty" yaml:"exec,omitempty"`
	Copy             *Copy             `json:"copy,omitempty" yaml:"copy,omitempty"`
	HTTP             *HTTP             `json:"http,omitempty" yaml:"http,omitempty"`
}
