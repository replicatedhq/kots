package applier

import (
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

func ensureResourcesPresent(input []byte) error {
	// TODO sort, order matters
	// TODO should we split multi-doc to retry on failed?

	fmt.Println("finding kubectl")
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return errors.Wrap(err, "failed to find kubectl")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	fmt.Println("applying manifest(s)")
	kubernetesApplier := NewKubectl(kubectl, config)
	go kubernetesApplier.Apply(input)

	return nil
}

func ensureResourcesMissing(input []byte) error {
	// TODO sort, order matters
	// TODO should we split multi-doc to retry on failed?

	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		fmt.Println(err)
		return err
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	kubernetesApplier := NewKubectl(kubectl, config)
	go kubernetesApplier.Remove(input)

	return nil
}
