package midstream

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	yaml "gopkg.in/yaml.v2"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	secretFilename = "./secret.yaml"
	objectsDir     = "./objects"
)

type WriteOptions struct {
	MidstreamDir string
	BaseDir      string
}

func (m *Midstream) WriteMidstream(options WriteOptions) error {
	_, err := os.Stat(options.MidstreamDir)
	if err == nil {
		// no error, the midstream already exists
		return nil
	}

	if err := os.MkdirAll(options.MidstreamDir, 0744); err != nil {
		return errors.Wrap(err, "failed to mkdir")
	}

	secretFilename, err := m.writePullSecret(options)
	if err != nil {
		return errors.Wrap(err, "failed to write kustomization")
	}

	if secretFilename != "" {
		m.Kustomization.Resources = append(m.Kustomization.Resources, secretFilename)
	}

	objectPatches, err := m.writeObjectsWithPullSecret(options)
	if err != nil {
		return errors.Wrap(err, "failed to write kustomization")
	}
	for _, patch := range objectPatches {
		m.Kustomization.PatchesStrategicMerge = append(m.Kustomization.PatchesStrategicMerge, kustomizetypes.PatchStrategicMerge(patch))
	}

	if err := m.writeKustomization(options); err != nil {
		return errors.Wrap(err, "failed to write kustomization")
	}

	return nil
}

func (m *Midstream) writeKustomization(options WriteOptions) error {
	relativeBaseDir, err := filepath.Rel(options.MidstreamDir, options.BaseDir)
	if err != nil {
		return errors.Wrap(err, "failed to determine relative path for base from midstream")
	}

	fileRenderPath := path.Join(options.MidstreamDir, "kustomization.yaml")

	m.Kustomization.Bases = []string{
		relativeBaseDir,
	}

	if err := k8sutil.WriteKustomizationToFile(m.Kustomization, fileRenderPath); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}

func (m *Midstream) writePullSecret(options WriteOptions) (string, error) {
	if m.PullSecret == nil {
		return "", nil
	}

	filename := filepath.Join(options.MidstreamDir, secretFilename)

	// TODO: Overwrite or not?
	_, err := os.Stat(filename)
	if err == nil {
		return secretFilename, nil
	}

	b, err := k8syaml.Marshal(m.PullSecret)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal pull secret")
	}

	if err := ioutil.WriteFile(filename, b, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write pull secret file")
	}

	return secretFilename, nil
}

func (m *Midstream) writeObjectsWithPullSecret(options WriteOptions) ([]string, error) {
	dir := filepath.Join(options.MidstreamDir, objectsDir)

	// TODO: Overwrite or not?
	// _, err := os.Stat(dir)
	// if err == nil {
	// 	return nil
	// }

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0744); err != nil {
			return nil, errors.Wrap(err, "failed to mkdir")
		}
	}

	resources := make([]string, 0)
	for _, o := range m.DocForPatches {
		err := func() error {
			resource := filepath.Join(objectsDir, fmt.Sprintf("%s.yaml", o.Metadata.Name))
			resources = append(resources, resource)

			filename := filepath.Join(options.MidstreamDir, resource)

			f, err := os.Create(filename)
			if err != nil {
				return errors.Wrap(err, "failed to craete resources file")
			}
			defer f.Close()

			withPullSecret := obejctWithPullSecret(o, m.PullSecret)

			b, err := yaml.Marshal(withPullSecret)
			if err != nil {
				return errors.Wrap(err, "failed to marshal object")
			}
			if _, err := f.Write(b); err != nil {
				return errors.Wrap(err, "failed to write object")
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return resources, nil
}

func obejctWithPullSecret(obj *k8sdoc.Doc, secret *corev1.Secret) *k8sdoc.Doc {
	return &k8sdoc.Doc{
		APIVersion: obj.APIVersion,
		Kind:       obj.Kind,
		Metadata: k8sdoc.Metadata{
			Name: obj.Metadata.Name,
		},
		Spec: k8sdoc.Spec{
			Template: k8sdoc.Template{
				Spec: k8sdoc.PodSpec{
					ImagePullSecrets: []k8sdoc.ImagePullSecret{
						{"name": "kotsadm-replicated-registry"},
					},
				},
			},
		},
	}
}
