// Note: This is a modified version of: https://github.com/dexidp/dex/blob/ed920dc27ad79c3593037ad658552e8e80bab928/storage/kubernetes/storage.go
package types

// KubernetesConfig values for the Kubernetes storage type.
type KubernetesConfig struct {
	InCluster      bool   `json:"inCluster"`
	KubeConfigFile string `json:"kubeConfigFile"`
}
