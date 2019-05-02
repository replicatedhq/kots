package patch

import (
	"bytes"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/k8sdeps"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
	"sigs.k8s.io/kustomize/pkg/target"
)

func (p *ShipPatcher) RunKustomize(kustomizationPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	fsys := fs.MakeRealFS()

	if err := p.runKustomize(buf, fsys, kustomizationPath); err != nil {
		return nil, errors.Wrap(err, "failed to run kustomize build")
	}

	return buf.Bytes(), nil
}

func (p *ShipPatcher) runKustomize(out io.Writer, fSys fs.FileSystem, kustomizationPath string) error {
	absPath, err := filepath.Abs(kustomizationPath)
	if err != nil {
		return err
	}

	ldr, err := loader.NewLoader(absPath, fSys)
	if err != nil {
		return errors.Wrap(err, "make loader")
	}

	k8sFactory := k8sdeps.NewFactory()

	kt, err := target.NewKustTarget(ldr, k8sFactory.ResmapF, k8sFactory.TransformerF)
	if err != nil {
		return errors.Wrap(err, "make customized kustomize target")
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return errors.Wrap(err, "make customized res map")
	}

	// Output the objects.
	res, err := allResources.EncodeAsYaml()
	if err != nil {
		return errors.Wrap(err, "encode as yaml")
	}
	_, err = out.Write(res)
	return err
}
