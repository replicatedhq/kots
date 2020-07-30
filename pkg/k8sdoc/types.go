package k8sdoc

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

type K8sDoc interface {
	PatchWithPullSecret(secret *corev1.Secret) K8sDoc
	ListImages() []string
}

type Doc struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type PodDoc struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       PodSpec  `yaml:"spec"`
}

type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

type Spec struct {
	Template    Template    `yaml:"template,omitempty"`
	JobTemplate JobTemplate `yaml:"jobTemplate,omitempty"`
}

type JobTemplate struct {
	Spec JobSpec `yaml:"spec"`
}

type JobSpec struct {
	Template Template `yaml:"template"`
}

type Template struct {
	Spec PodSpec `yaml:"spec"`
}

type PodSpec struct {
	Containers       []Container       `yaml:"containers,omitempty"`     // don't write empty array into patches
	InitContainers   []Container       `yaml:"initContainers,omitempty"` // don't write empty array into patches
	ImagePullSecrets []ImagePullSecret `yaml:"imagePullSecrets"`
}

type ImagePullSecret map[string]string

type Container struct {
	Image string `yaml:"image"`
}

func ParseYAML(yamlDoc []byte) (K8sDoc, error) {
	doc := &Doc{}
	if err := yaml.Unmarshal(yamlDoc, doc); err != nil {
		return nil, errors.Wrap(err, "failed to parse yaml")
	}

	if doc.Kind != "Pod" {
		return doc, nil
	}

	podDoc := &PodDoc{}
	if err := yaml.Unmarshal(yamlDoc, podDoc); err != nil {
		return nil, errors.Wrap(err, "failed to parse yaml")
	}
	return podDoc, nil
}

func (d *Doc) PatchWithPullSecret(secret *corev1.Secret) K8sDoc {
	newObj := &Doc{
		APIVersion: d.APIVersion,
		Kind:       d.Kind,
		Metadata: Metadata{
			Name:      d.Metadata.Name,
			Namespace: d.Metadata.Namespace,
			Labels:    d.Metadata.Labels,
		},
	}
	switch d.Kind {
	case "CronJob":
		newObj.Spec = Spec{
			JobTemplate: JobTemplate{
				Spec: JobSpec{
					Template: Template{
						Spec: PodSpec{
							ImagePullSecrets: []ImagePullSecret{
								{"name": "kotsadm-replicated-registry"},
							},
						},
					},
				},
			},
		}

	default:
		newObj.Spec = Spec{
			Template: Template{
				Spec: PodSpec{
					ImagePullSecrets: []ImagePullSecret{
						{"name": "kotsadm-replicated-registry"},
					},
				},
			},
		}
	}

	return newObj
}

func (d *Doc) ListImages() []string {
	images := make([]string, 0)
	for _, container := range d.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	for _, container := range d.Spec.Template.Spec.InitContainers {
		images = append(images, container.Image)
	}

	for _, container := range d.Spec.JobTemplate.Spec.Template.Spec.InitContainers {
		images = append(images, container.Image)
	}

	for _, container := range d.Spec.JobTemplate.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}
	return images
}

func (d *PodDoc) PatchWithPullSecret(secret *corev1.Secret) K8sDoc {
	return &PodDoc{
		APIVersion: d.APIVersion,
		Kind:       d.Kind,
		Metadata: Metadata{
			Name:      d.Metadata.Name,
			Namespace: d.Metadata.Namespace,
			Labels:    d.Metadata.Labels,
		},
		Spec: PodSpec{
			ImagePullSecrets: []ImagePullSecret{
				{"name": "kotsadm-replicated-registry"},
			},
		},
	}
}

func (d *PodDoc) ListImages() []string {
	images := make([]string, 0)
	for _, container := range d.Spec.Containers {
		images = append(images, container.Image)
	}

	for _, container := range d.Spec.InitContainers {
		images = append(images, container.Image)
	}
	return images
}
