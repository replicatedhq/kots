package cluster

import (
	"fmt"
	"os/exec"

	"github.com/replicatedhq/ship-cluster/ship-cd/pkg/controller/cluster/applier"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func (r *ReconcileCluster) ensureResourcesPresent(input []byte) error {
	// TODO sort, order matters
	// TODO should we split multi-doc to retry on failed?

	fmt.Println("finding kubectl")
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("getting kubeconfig")
	restClientConfig, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
		return err
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	fmt.Println("applying manifest(s)")
	kubernetesApplier := applier.NewKubectl(kubectl, restClientConfig)
	go kubernetesApplier.Apply(input)

	return nil
}

func (r *ReconcileCluster) ensureResourcesMissing(input []byte) error {
	// TODO sort, order matters
	// TODO should we split multi-doc to retry on failed?

	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		fmt.Println(err)
		return err
	}

	restClientConfig, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
		return err
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	kubernetesApplier := applier.NewKubectl(kubectl, restClientConfig)
	go kubernetesApplier.Remove(input)

	return nil
}
