package types

type KurlNodes struct {
	Nodes         []Node `json:"nodes"`
	HA            bool   `json:"ha"`
	IsKurlEnabled bool   `json:"isKurlEnabled"`
}

type Node struct {
	Name           string            `json:"name"`
	IsConnected    bool              `json:"isConnected"`
	IsMasterNode   bool              `json:"isMasterNode"`
	CanDelete      bool              `json:"canDelete"`
	KubeletVersion string            `json:"kubeletVersion"`
	CPU            CapacityAvailable `json:"cpu"`
	Memory         CapacityAvailable `json:"memory"`
	Pods           CapacityAvailable `json:"pods"`
	Conditions     NodeConditions    `json:"conditions"`
}

type CapacityAvailable struct {
	Capacity  float64 `json:"capacity"`
	Available float64 `json:"available"`
}

type NodeConditions struct {
	MemoryPressure bool `json:"memoryPressure"`
	DiskPressure   bool `json:"diskPressure"`
	PidPressure    bool `json:"pidPressure"`
	Ready          bool `json:"ready"`
}
