package types

const EMBEDDED_CLUSTER_LABEL = "kots.io/embedded-cluster"
const EMBEDDED_CLUSTER_ROLE_LABEL = EMBEDDED_CLUSTER_LABEL + "-role"

type HelmVMNodes struct {
	Nodes           []Node `json:"nodes"`
	HA              bool   `json:"ha"`
	IsHelmVMEnabled bool   `json:"isHelmVMEnabled"`
}

type Node struct {
	Name             string         `json:"name"`
	IsConnected      bool           `json:"isConnected"`
	IsReady          bool           `json:"isReady"`
	IsPrimaryNode    bool           `json:"isPrimaryNode"`
	CanDelete        bool           `json:"canDelete"`
	KubeletVersion   string         `json:"kubeletVersion"`
	KubeProxyVersion string         `json:"kubeProxyVersion"`
	OperatingSystem  string         `json:"operatingSystem"`
	KernelVersion    string         `json:"kernelVersion"`
	CPU              CapacityUsed   `json:"cpu"`
	Memory           CapacityUsed   `json:"memory"`
	Pods             CapacityUsed   `json:"pods"`
	Labels           []string       `json:"labels"`
	Conditions       NodeConditions `json:"conditions"`
	PodList          []PodInfo      `json:"podList"`
}

type CapacityUsed struct {
	Capacity float64 `json:"capacity"`
	Used     float64 `json:"used"`
}

type NodeConditions struct {
	MemoryPressure bool `json:"memoryPressure"`
	DiskPressure   bool `json:"diskPressure"`
	PidPressure    bool `json:"pidPressure"`
	Ready          bool `json:"ready"`
}

type PodInfo struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
	CPU       string `json:"cpu"`
	Memory    string `json:"memory"`
}
