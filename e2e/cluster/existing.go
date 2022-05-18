package cluster

type Existing struct {
	kubeconfig string
}

func NewExisting(kubeconfig string) *Existing {
	return &Existing{
		kubeconfig: kubeconfig,
	}
}

func (c *Existing) GetKubeconfig() string {
	return c.kubeconfig
}

func (c *Existing) Teardown() {
	// nothing to do
}
