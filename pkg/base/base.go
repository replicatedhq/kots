package base

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"gopkg.in/yaml.v2"
	batchv1 "k8s.io/api/batch/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type Base struct {
	Path            string
	Namespace       string
	Files           []BaseFile
	ErrorFiles      []BaseFile
	AdditionalFiles []BaseFile
	Bases           []Base
}

func (in *Base) DeepCopyInto(out *Base) {
	*out = *in
	if in.Files != nil {
		out.Files = make([]BaseFile, len(in.Files))
		for i, file := range in.Files {
			out.Files[i] = *file.DeepCopy()
		}
	}
	if in.ErrorFiles != nil {
		out.ErrorFiles = make([]BaseFile, len(in.ErrorFiles))
		for i, file := range in.ErrorFiles {
			out.ErrorFiles[i] = *file.DeepCopy()
		}
	}
	if in.AdditionalFiles != nil {
		out.AdditionalFiles = make([]BaseFile, len(in.AdditionalFiles))
		for i, file := range in.AdditionalFiles {
			out.AdditionalFiles[i] = *file.DeepCopy()
		}
	}
	if in.Bases != nil {
		out.Bases = make([]Base, len(in.Bases))
		for i, base := range in.Bases {
			out.Bases[i] = *base.DeepCopy()
		}
	}
	return
}

func (in *Base) DeepCopy() *Base {
	if in == nil {
		return nil
	}
	out := new(Base)
	in.DeepCopyInto(out)
	return out
}

func (b *Base) SetNamespace(namespace string) {
	b.Namespace = namespace
	for i := range b.Bases {
		b.Bases[i].SetNamespace(namespace)
	}
}

type BaseFile struct {
	Path    string
	Content []byte
	Error   error
}

func (in *BaseFile) DeepCopyInto(out *BaseFile) {
	*out = *in
}

func (in *BaseFile) DeepCopy() *BaseFile {
	if in == nil {
		return nil
	}
	out := new(BaseFile)
	in.DeepCopyInto(out)
	return out
}

type OverlySimpleGVK struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name        string                 `yaml:"name"`
	Namespace   string                 `yaml:"namespace"`
	Annotations map[string]interface{} `json:"annotations"`
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func GetGVKWithNameAndNs(content []byte, baseNS string) (string, OverlySimpleGVK) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return "", o
	}

	namespace := baseNS
	if o.Metadata.Namespace != "" {
		namespace = o.Metadata.Namespace
	}

	return fmt.Sprintf("%s-%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name, namespace), o
}

func (f *BaseFile) transpileHelmHooksToKotsHooks() error {
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

type ParseError struct {
	Err error
}

func (e ParseError) Error() string {
	return e.Err.Error()
}

// ShouldBeIncludedInBaseKustomization attempts to determine if this is a valid Kubernetes manifest.
// It accomplished this by trying to unmarshal the YAML and looking for a apiVersion and Kind
func (f BaseFile) ShouldBeIncludedInBaseKustomization(excludeKotsKinds bool) (bool, error) {
	var m interface{}

	if err := yaml.Unmarshal(f.Content, &m); err != nil {
		// check if this is a yaml file
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			return false, ParseError{Err: err}
		}
		return false, nil
	}

	o := OverlySimpleGVK{}
	_ = yaml.Unmarshal(f.Content, &o) // error should be caught in previous unmarshal

	// check if this is a kubernetes document
	if o.APIVersion == "" || o.Kind == "" {
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			// ignore empty files and files with only comments
			if m == nil {
				return false, nil
			}
			return false, ParseError{Err: errors.New("not a kubernetes document")}
		}
		return false, nil
	}

	// Backup is never deployed. kots.io/exclude and kots.io/when are used to enable snapshots
	if excludeKotsKinds {
		if kotsutil.IsKotsKind(o.APIVersion, o.Kind) {
			return false, nil
		}
	}

	exclude, err := isExcludedByAnnotation(o.Metadata.Annotations)
	return !exclude, errors.Wrapf(err, "failed to check if object %s, kind %s/%s is excluded by annotation", o.Metadata.Name, o.APIVersion, o.Kind)
}

func isExcludedByAnnotation(annotations map[string]interface{}) (bool, error) {
	if annotations == nil {
		return false, nil
	}

	if val, ok := annotations["kots.io/exclude"]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal, nil
		}

		if strVal, ok := val.(string); ok {
			boolVal, err := strconv.ParseBool(strVal)
			if err != nil {
				// should this be a ParseError?
				return false, errors.Errorf("unable to parse %s as bool in exclude annotation", strVal)
			}

			return boolVal, nil
		}

		// should this be a ParseError?
		return false, errors.Errorf("unexpected type in exclude annotation: %T", val)
	}

	if val, ok := annotations["kots.io/when"]; ok {
		if boolVal, ok := val.(bool); ok {
			return !boolVal, nil
		}

		if strVal, ok := val.(string); ok {
			boolVal, err := strconv.ParseBool(strVal)
			if err != nil {
				// should this be a ParseError?
				return false, errors.Errorf("unable to parse %s as bool in when annotation", strVal)
			}

			return !boolVal, nil
		}

		// should this be a ParseError?
		return false, errors.Errorf("unexpected type in when annotation: %T", val)
	}

	return false, nil
}

func (f BaseFile) IsKotsKind() (bool, error) {
	var m interface{}

	if err := yaml.Unmarshal(f.Content, &m); err != nil {
		// check if this is a yaml file
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			return false, ParseError{Err: err}
		}
		return false, nil
	}

	o := OverlySimpleGVK{}
	_ = yaml.Unmarshal(f.Content, &o) // error should be caught in previous unmarshal

	// check if this is a kubernetes document
	if o.APIVersion == "" || o.Kind == "" {
		// check if this is a yaml file
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			// ignore empty files and files with only comments
			if m == nil {
				return false, nil
			}
			return false, ParseError{Err: errors.New("not a kubernetes document")}
		}
		return false, nil
	}

	return kotsutil.IsKotsKind(o.APIVersion, o.Kind), nil
}

func (b Base) ListErrorFiles() []BaseFile {
	files := append([]BaseFile{}, b.ErrorFiles...)
	for _, b := range b.Bases {
		files = append(files, PrependBaseFilesPath(b.ListErrorFiles(), b.Path)...)
	}
	return files
}

func PrependBaseFilesPath(files []BaseFile, prefix string) []BaseFile {
	if prefix == "" {
		return files
	}
	next := []BaseFile{}
	for _, file := range files {
		file.Path = path.Join(prefix, file.Path)
		next = append(next, file)
	}
	return next
}

func FindImages(b *Base) ([]string, []k8sdoc.K8sDoc, error) {
	uniqueImages := make(map[string]bool)
	objectsWithImages := make([]k8sdoc.K8sDoc, 0) // all objects where images are referenced from

	for _, file := range b.Files {
		parsed, err := k8sdoc.ParseYAML(file.Content)
		if err != nil {
			continue
		}

		images := parsed.ListImages()
		if len(images) > 0 {
			objectsWithImages = append(objectsWithImages, parsed)
		}

		for _, image := range images {
			uniqueImages[image] = true
		}
	}

	for _, subBase := range b.Bases {
		subImages, subObjects, err := FindImages(&subBase)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find images in sub base %s", subBase.Path)
		}

		objectsWithImages = append(objectsWithImages, subObjects...)

		for _, subImage := range subImages {
			uniqueImages[subImage] = true
		}
	}

	result := make([]string, 0, len(uniqueImages))
	for i := range uniqueImages {
		result = append(result, i)
	}
	sort.Strings(result) // sort the images to get an ordered and reproducible output for easier testing

	return result, objectsWithImages, nil
}
