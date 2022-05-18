package cluster

type Interface interface {
	GetKubeconfig() string
	Teardown()
}
