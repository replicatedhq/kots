package base

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"gopkg.in/yaml.v2"
	batchv1 "k8s.io/api/batch/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type Base struct {
	Files []BaseFile
}

type BaseFile struct {
	Path    string
	Content []byte
}

type OverlySimpleGVK struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name        string                 `yaml:"name"`
	Annotations map[string]interface{} `json:"annotations"`
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func GetGVKWithNameHash(content []byte) []byte {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return nil
	}

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name)))
	return h.Sum(nil)
}

func (f BaseFile) transpileHelmHooksToKotsHooks() error {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(f.Content, nil, nil)
	if err != nil {
		return nil // this isn't an error, it's just not a job witih a hook, that's certain
	}

	// we currently only support hooks on jobs
	if gvk.Group != "batch" || gvk.Version != "v1" || gvk.Kind != "Job" {
		return nil
	}

	job := obj.(*batchv1.Job)

	helmHookDeletePolicyAnnotation, ok := job.Annotations["helm.sh/hook-delete-policy"]
	if !ok {
		return nil
	}

	job.Annotations["kots.io/hook-delete-policy"] = helmHookDeletePolicyAnnotation

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(job, &b); err != nil {
		return errors.Wrap(err, "failed to encode job")
	}

	f.Content = b.Bytes()
	return nil
}

// ShouldBeIncludedInBaseKustomization attempts to determine if this is a valid Kubernetes manifest.
// It accomplished this by trying to unmarshal the YAML and looking for a apiVersion and Kind
func (f BaseFile) ShouldBeIncludedInBaseKustomization(excludeKotsKinds bool) (bool, error) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(f.Content, &o); err != nil {
		return false, nil
	}

	if o.APIVersion == "" || o.Kind == "" {
		return false, nil
	}

	if o.Metadata.Annotations != nil {
		if val, ok := o.Metadata.Annotations["kots.io/exclude"]; ok {
			if boolVal, ok := val.(bool); ok {
				return !boolVal, nil
			}

			if strVal, ok := val.(string); ok {
				boolVal, err := strconv.ParseBool(strVal)
				if err != nil {
					return true, errors.Errorf("unable to parse %s as bool in exclude annotation of object %s, kind %s/%s", strVal, o.Metadata.Name, o.APIVersion, o.Kind)
				}

				return !boolVal, nil
			}

			return true, errors.Errorf("unexpected type in exclude annotation of %s/%s: %T", o.APIVersion, o.Metadata.Name, val)
		}

		if val, ok := o.Metadata.Annotations["kots.io/when"]; ok {
			if boolVal, ok := val.(bool); ok {
				return boolVal, nil
			}

			if strVal, ok := val.(string); ok {
				boolVal, err := strconv.ParseBool(strVal)
				if err != nil {
					return true, errors.Errorf("unable to parse %s as bool in wen annotation of object %s, kind %s/%s", strVal, o.Metadata.Name, o.APIVersion, o.Kind)
				}

				return boolVal, nil
			}

			return true, errors.Errorf("unexpected type in when annotation of %s/%s: %T", o.APIVersion, o.Metadata.Name, val)
		}
	}
	if excludeKotsKinds {
		if o.APIVersion == "kots.io/v1beta1" {
			return false, nil
		}

		if o.APIVersion == "troubleshoot.replicated.com/v1beta1" {
			return false, nil
		}

		// In addition to kotskinds, we exclude the application crd for now
		if o.APIVersion == "app.k8s.io/v1beta1" {
			return false, nil
		}

		if o.APIVersion == "velero.io/v1" && o.Kind == "Backup" {
			return false, nil
		}
	}

	return true, nil
}
