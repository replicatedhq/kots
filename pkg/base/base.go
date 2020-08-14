package base

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/logger"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

var metadataAccessor = meta.NewAccessor()

type Base struct {
	Path       string
	Namespace  string
	Files      []BaseFile
	ErrorFiles []BaseFile
	Bases      []Base
}

type BaseFile struct {
	Path       string
	Content    []byte
	Error      error
	HookEvents []HookEvent
}

type OverlySimpleGVK struct {
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Metadata   OverlySimpleMetadata `json:"metadata"`
}

type OverlySimpleMetadata struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Annotations map[string]interface{} `json:"annotations"`
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

func GetGVKWithNameAndNs(content []byte, baseNS string) string {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return ""
	}

	namespace := baseNS
	if o.Metadata.Namespace != "" {
		namespace = o.Metadata.Namespace
	}

	return fmt.Sprintf("%s-%s-%s-%s", o.APIVersion, o.Kind, o.Metadata.Name, namespace)
}

type ParseError struct {
	Err error
}

func (e ParseError) Error() string {
	return e.Err.Error()
}

// ShouldBeIncludedInBaseKustomization attempts to determine if this is a valid Kubernetes manifest.
// It accomplished this by trying to unmarshal the YAML and looking for a apiVersion and Kind
func (f BaseFile) ShouldBeIncludedInBaseKustomization(excludeKotsKinds bool, log *logger.Logger) (bool, error) {
	// +++ preserve backwards compatibility
	// the next 20 lines are all to make up for the fact that we allowed annotation values to be booleans in kots kinds

	o := OverlySimpleGVK{}
	_ = yaml.Unmarshal(f.Content, &o) // error should be caught by decode

	gv, err := schema.ParseGroupVersion(o.APIVersion)
	if err == nil {
		if o.APIVersion != "" && o.Kind != "" {
			gvk := &schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    o.Kind,
			}
			if excludeKotsKinds {
				if isKotsAPIVersionKind(gvk) {
					return false, nil
				}
			}
		}
	}

	if exclude, _ := isExcludedByAnnotationCompat(o.Metadata.Annotations); exclude {
		return false, nil
	}

	// --- preserve backwards compatibility

	obj, gvk, err := f.maybeDecode()
	if err != nil || gvk == nil {
		return false, err
	}

	name, _ := metadataAccessor.Name(obj)
	annotations, _ := metadataAccessor.Annotations(obj)

	// Backup is never deployed. kots.io/exclude and kots.io/when are used to enable snapshots
	if excludeKotsKinds {
		if isKotsAPIVersionKind(gvk) {
			return false, nil
		}
	}

	exclude, err := isExcludedByAnnotation(annotations)
	if err != nil {
		// preserve backwards compatibility
		if log != nil {
			log.Error(fmt.Errorf("Failed to check kots.io exclude annotations of object %s kind %s: %v", name, gvk, err))
		}
	}
	if exclude {
		return false, nil
	}

	return true, nil
}

func (f BaseFile) GetKotsHookEvents() ([]HookEvent, error) {
	obj, gvk, err := f.maybeDecode()
	if err != nil || gvk == nil {
		return nil, err
	}

	annotations, _ := metadataAccessor.Annotations(obj)

	return getKotsHookEvents(annotations), nil
}

func (f BaseFile) IsKotsKind() (bool, error) {
	// +++ preserve backwards compatibility
	// the next 20 lines are all to make up for the fact that we allowed annotation values to be booleans in kots kinds

	o := OverlySimpleGVK{}
	_ = yaml.Unmarshal(f.Content, &o) // error should be caught by decode

	gv, err := schema.ParseGroupVersion(o.APIVersion)
	if err == nil {
		if o.APIVersion != "" && o.Kind != "" {
			gvk := &schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    o.Kind,
			}
			if isKotsAPIVersionKind(gvk) {
				return true, nil
			}
		}
	}

	// --- preserve backwards compatibility

	_, gvk, err := f.maybeDecode()
	if err != nil || gvk == nil {
		return false, err
	}

	return isKotsAPIVersionKind(gvk), nil
}

func (f BaseFile) maybeDecode() (runtime.Object, *schema.GroupVersionKind, error) {
	var m interface{}

	if err := yaml.Unmarshal(f.Content, &m); err != nil {
		// check if this is a yaml file
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			return nil, nil, ParseError{Err: err}
		}
		return nil, nil, nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(f.Content, nil, nil)
	// check if this is a kubernetes document
	if err != nil {
		if ext := filepath.Ext(f.Path); ext == ".yaml" || ext == ".yml" {
			// ignore empty files and files with only comments
			if m == nil {
				return nil, nil, nil
			}
			return nil, nil, ParseError{Err: errors.New("not a kubernetes document")}
		}
		return nil, nil, nil
	}
	return obj, gvk, err
}

func isKotsAPIVersionKind(gvk *schema.GroupVersionKind) bool {
	if gvk.Group == "velero.io" && gvk.Kind == "Backup" {
		return true
	}
	if gvk.Group == "kots.io" {
		return true
	}
	if gvk.Group == "troubleshoot.replicated.com" {
		return true
	}
	// In addition to kotskinds, we exclude the application crd for now
	if gvk.Group == "app.k8s.io" {
		return true
	}
	return false
}

func isExcludedByAnnotation(annotations map[string]string) (bool, error) {
	var retErr error

	if strVal, ok := annotations["kots.io/exclude"]; ok {
		boolVal, err := strconv.ParseBool(strVal)
		if err != nil {
			// should this be a ParseError?
			retErr = multierr.Append(retErr, errors.Errorf("failed to parse %s as bool in kots.io/exclude annotation", strVal))
		} else if boolVal {
			return true, retErr
		}
	}

	if strVal, ok := annotations["kots.io/when"]; ok {
		boolVal, err := strconv.ParseBool(strVal)
		if err != nil {
			// should this be a ParseError?
			retErr = multierr.Append(retErr, errors.Errorf("failed to parse %s as bool in kots.io/when annotation", strVal))
		} else if !boolVal {
			return true, retErr
		}
	}

	return false, retErr
}

func isExcludedByAnnotationCompat(annotations map[string]interface{}) (bool, error) {
	var retErr error

	if boolVal, ok := annotations["kots.io/exclude"].(bool); ok {
		if boolVal {
			return true, retErr
		}
	}

	if boolVal, ok := annotations["kots.io/when"].(bool); ok {
		if !boolVal {
			return true, retErr
		}
	}

	return false, retErr
}

func hasKotsHookEvents(annotations map[string]string) bool {
	return len(getKotsHookEvents(annotations)) > 0
}

func getKotsHookEvents(annotations map[string]string) []HookEvent {
	events := []HookEvent{}
	for _, hookType := range strings.Split(annotations[HookAnnotation], ",") {
		hookType = strings.ToLower(strings.TrimSpace(hookType))
		e, ok := hookEvents[hookType]
		if ok {
			events = append(events, e)
		}
	}
	return events
}

func (b Base) ListErrorFiles() []BaseFile {
	files := append([]BaseFile{}, b.ErrorFiles...)
	for _, b := range b.Bases {
		files = append(files, b.ListErrorFiles()...)
	}
	return files
}
