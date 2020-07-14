package midstream

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	secretFilename  = "secret.yaml"
	patchesFilename = "pullsecrets.yaml"
)

type WriteOptions struct {
	MidstreamDir string
	BaseDir      string
	AppSlug      string
	AppSequence  int64
	IsGitOps     bool
}

func (m *Midstream) KustomizationFilename(options WriteOptions) string {
	return path.Join(options.MidstreamDir, "kustomization.yaml")
}

func (m *Midstream) WriteMidstream(options WriteOptions) error {
	var existingKustomization *kustomizetypes.Kustomization

	_, err := os.Stat(m.KustomizationFilename(options))
	if err == nil {
		k, err := k8sutil.ReadKustomizationFromFile(m.KustomizationFilename(options))
		if err != nil {
			return errors.Wrap(err, "load existing kustomization")
		}
		existingKustomization = k
	}

	if err := os.MkdirAll(options.MidstreamDir, 0744); err != nil {
		return errors.Wrap(err, "failed to mkdir")
	}

	secretFilename, err := m.writePullSecret(options)
	if err != nil {
		return errors.Wrap(err, "failed to write secret")
	}

	if secretFilename != "" {
		m.Kustomization.Resources = append(m.Kustomization.Resources, secretFilename)
	}

	if err := m.writeObjectsWithPullSecret(options); err != nil {
		return errors.Wrap(err, "failed to write patches")
	}

	m.mergeKustomization(existingKustomization)

	if err := m.writeKustomization(options); err != nil {
		return errors.Wrap(err, "failed to write kustomization")
	}

	return nil
}

func (m *Midstream) mergeKustomization(existing *kustomizetypes.Kustomization) {
	if existing == nil {
		return
	}

	filteredImages := removeExistingImages(m.Kustomization.Images, existing.Images)
	m.Kustomization.Images = append(m.Kustomization.Images, filteredImages...)

	existing.PatchesStrategicMerge = removeFromPatches(existing.PatchesStrategicMerge, patchesFilename)
	newPatches := findNewPatches(m.Kustomization.PatchesStrategicMerge, existing.PatchesStrategicMerge)
	m.Kustomization.PatchesStrategicMerge = append(existing.PatchesStrategicMerge, newPatches...)

	newResources := findNewStrings(m.Kustomization.Resources, existing.Resources)
	m.Kustomization.Resources = append(existing.Resources, newResources...)

	delete(existing.CommonAnnotations, "kots.io/app-slug")
	delete(existing.CommonAnnotations, "kots.io/app-sequence")
	m.Kustomization.CommonAnnotations = mergeMaps(m.Kustomization.CommonAnnotations, existing.CommonAnnotations)
}

func mergeMaps(new map[string]string, existing map[string]string) map[string]string {
	merged := existing
	if merged == nil {
		merged = make(map[string]string)
	}
	for key, value := range new {
		merged[key] = value
	}
	return merged
}

func (m *Midstream) writeKustomization(options WriteOptions) error {
	relativeBaseDir, err := filepath.Rel(options.MidstreamDir, options.BaseDir)
	if err != nil {
		return errors.Wrap(err, "failed to determine relative path for base from midstream")
	}

	fileRenderPath := m.KustomizationFilename(options)

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

	absFilename := filepath.Join(options.MidstreamDir, secretFilename)

	b, err := k8syaml.Marshal(m.PullSecret)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal pull secret")
	}

	if err := ioutil.WriteFile(absFilename, b, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write pull secret file")
	}

	return secretFilename, nil
}

func (m *Midstream) writeObjectsWithPullSecret(options WriteOptions) error {
	filename := filepath.Join(options.MidstreamDir, patchesFilename)
	if len(m.DocForPatches) == 0 {
		err := os.Remove(filename)
		if err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to delete pull secret patches")
		}

		return nil
	}

	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "failed to create resources file")
	}
	defer f.Close()

	for _, o := range m.DocForPatches {
		withPullSecret := obejctWithPullSecret(o, m.PullSecret)

		b, err := yaml.Marshal(withPullSecret)
		if err != nil {
			return errors.Wrap(err, "failed to marshal object")
		}

		if _, err := f.Write([]byte("---\n")); err != nil {
			return errors.Wrap(err, "failed to write object")
		}
		if _, err := f.Write(b); err != nil {
			return errors.Wrap(err, "failed to write object")
		}
	}

	m.Kustomization.PatchesStrategicMerge = append(m.Kustomization.PatchesStrategicMerge, patchesFilename)

	return nil
}

func removeFromPatches(patches []kustomizetypes.PatchStrategicMerge, filename string) []kustomizetypes.PatchStrategicMerge {
	newPatches := []kustomizetypes.PatchStrategicMerge{}
	for _, patch := range patches {
		if string(patch) != filename {
			newPatches = append(newPatches, patch)
		}
	}
	return newPatches
}

func obejctWithPullSecret(obj *k8sdoc.Doc, secret *corev1.Secret) *k8sdoc.Doc {
	newObj := &k8sdoc.Doc{
		APIVersion: obj.APIVersion,
		Kind:       obj.Kind,
		Metadata: k8sdoc.Metadata{
			Name:      obj.Metadata.Name,
			Namespace: obj.Metadata.Namespace,
			Labels:    obj.Metadata.Labels,
		},
	}
	switch obj.Kind {
	case "CronJob":
		newObj.Spec = k8sdoc.Spec{
			JobTemplate: k8sdoc.JobTemplate{
				Spec: k8sdoc.JobSpec{
					Template: k8sdoc.Template{
						Spec: k8sdoc.PodSpec{
							ImagePullSecrets: []k8sdoc.ImagePullSecret{
								{"name": "kotsadm-replicated-registry"},
							},
						},
					},
				},
			},
		}

	default:
		newObj.Spec = k8sdoc.Spec{
			Template: k8sdoc.Template{
				Spec: k8sdoc.PodSpec{
					ImagePullSecrets: []k8sdoc.ImagePullSecret{
						{"name": "kotsadm-replicated-registry"},
					},
				},
			},
		}
	}

	return newObj
}
