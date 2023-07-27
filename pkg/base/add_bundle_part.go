package base

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
)

func AddBundlePart(baseDir string, filename string, content []byte) error {
	_, err := os.Stat(path.Join(baseDir, "admin-console", filename))
	if err == nil {
		return errors.New("base bundle file already exists")
	}

	if err := os.WriteFile(path.Join(baseDir, "admin-console", filename), content, 0644); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	k, err := k8sutil.ReadKustomizationFromFile(path.Join(baseDir, "kustomization.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read kustomization file")
	}

	k.Resources = append(k.Resources, path.Join("admin-console", filename))

	if err := k8sutil.WriteKustomizationToFile(*k, path.Join(baseDir, "kustomization.yaml")); err != nil {
		return errors.Wrap(err, "failed to write kustomiation file")
	}

	return nil
}
