package redact

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/redact/types"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

type RedactorMetadata struct {
	Metadata types.RedactorList `json:"metadata"`

	Redact string `json:"redact"`
}

const (
	redactConfigMapName     = "kotsadm-redact"
	redactSpecConfigMapName = "kotsadm-redact-spec"
	redactSpecDataKey       = "redact-spec"
)

func GetKotsadmRedactSpecURI() string {
	return fmt.Sprintf("configmap/%s/%s/%s", util.PodNamespace, redactSpecConfigMapName, redactSpecDataKey)
}

// GenerateKotsadmRedactSpec creates a configmap that contains the admin console custom redaction yaml spec
// generated from "kotsadm-redact" configmap for collecting support bundles. contains the full redact spec type that is supported by troubleshoot.
func GenerateKotsadmRedactSpec(clientset kubernetes.Interface) error {
	spec, _, err := GetRedactSpec()
	if err != nil {
		return errors.Wrap(err, "failed to get redact spec")
	}

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), redactSpecConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read redact spec configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		configmap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      redactSpecConfigMapName,
				Namespace: util.PodNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{
				redactSpecDataKey: spec,
			},
		}

		_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), configmap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create redactor spec configmap")
		}

		return nil
	}

	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}
	existingConfigMap.Data[redactSpecDataKey] = spec
	existingConfigMap.ObjectMeta.Labels = kotsadmtypes.GetKotsadmLabels()

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update redactor spec secret")
	}

	return nil
}

// GetRedactSpec returns the redaction yaml spec, a pretty error string, and the underlying error
func GetRedactSpec() (string, string, error) {
	configMap, errstr, err := getRedactConfigmap()
	if err != nil || configMap == nil {
		return "", errstr, errors.Wrap(err, "get redactors configmap")
	}

	return getRedactSpec(configMap)
}

func getRedactSpec(configMap *v1.ConfigMap) (string, string, error) {
	redactObj, err := buildFullRedact(configMap)
	if err != nil {
		return "", "failed to build full redact yaml", err
	}

	yamlBytes, err := util.MarshalIndent(2, redactObj)
	if err != nil {
		return "", "failed to render full redact yaml", err
	}
	return string(yamlBytes), "", nil
}

func GetRedact() (*troubleshootv1beta2.Redactor, error) {
	configmap, _, err := getRedactConfigmap()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors configmap")
	}
	if configmap == nil {
		return nil, nil
	}

	return buildFullRedact(configmap)
}

func GetRedactInfo() ([]types.RedactorList, error) {
	configmap, _, err := getRedactConfigmap()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors configmap")
	}
	if configmap == nil {
		return nil, nil
	}

	if combinedYaml, ok := configmap.Data["kotsadm-redact"]; ok {
		// this is the key used for the combined redact list, so run the migration
		newMap, err := splitRedactors(combinedYaml)
		if err != nil {
			return nil, errors.Wrap(err, "failed to split combined redactors")
		}
		configmap.Data = newMap

		// now that the redactors have been split, save the configmap
		configmap, err = writeRedactConfigmap(configmap)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update configmap")
		}
	}

	list := []types.RedactorList{}

	for k, v := range configmap.Data {
		redactorEntry := RedactorMetadata{}
		err = json.Unmarshal([]byte(v), &redactorEntry)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse key %s", k)
		}
		list = append(list, redactorEntry.Metadata)
	}
	return list, nil
}

func GetRedactBySlug(slug string) (*RedactorMetadata, error) {
	configmap, _, err := getRedactConfigmap()
	if err != nil {
		return nil, err
	}
	if configmap == nil {
		return nil, errors.Wrap(err, "get redactors configmap")
	}

	redactString, ok := configmap.Data[slug]
	if !ok {
		return nil, fmt.Errorf("redactor %s not found", slug)
	}

	redactorEntry := RedactorMetadata{}
	err = json.Unmarshal([]byte(redactString), &redactorEntry)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse redactor %s", slug)
	}

	return &redactorEntry, nil
}

// SetRedactSpec sets the global redact spec to the specified string, and returns a pretty error string + the underlying error
func SetRedactSpec(spec string) (string, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return "failed to get k8s clientset", errors.Wrap(err, "failed to get k8s clientset")
	}

	configMap, errMsg, err := getRedactConfigmap()
	if err != nil {
		return errMsg, errors.Wrap(err, "get redactors configmap")
	}

	newMap, err := splitRedactors(spec)
	if err != nil {
		return "failed to split redactors", errors.Wrap(err, "failed to split redactors")
	}

	configMap.Data = newMap
	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return "failed to update kotsadm-redact configMap", errors.Wrap(err, "failed to update kotsadm-redact configMap")
	}
	return "", nil
}

// updates/creates an individual redact with the provided metadata and yaml
func SetRedactYaml(slug, description string, enabled, newRedact bool, yamlBytes []byte) (*RedactorMetadata, error) {
	configMap, _, err := getRedactConfigmap()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors configmap")
	}

	newData, redactorEntry, err := setRedactYaml(slug, description, enabled, newRedact, time.Now(), yamlBytes, configMap.Data)
	if err != nil {
		return nil, err
	}

	configMap.Data = newData

	_, err = writeRedactConfigmap(configMap)
	if err != nil {
		return nil, errors.Wrapf(err, "write configMap with updated redact")
	}
	return redactorEntry, nil
}

// sets whether an individual redactor is enabled
func SetRedactEnabled(slug string, enabled bool) (*RedactorMetadata, error) {
	configMap, _, err := getRedactConfigmap()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors configmap")
	}

	newData, redactorEntry, err := setRedactEnabled(slug, enabled, time.Now(), configMap.Data)
	if err != nil {
		return nil, err
	}

	configMap.Data = newData

	_, err = writeRedactConfigmap(configMap)
	if err != nil {
		return nil, errors.Wrapf(err, "write configMap with updated redact")
	}
	return redactorEntry, nil
}

func setRedactEnabled(slug string, enabled bool, currentTime time.Time, data map[string]string) (map[string]string, *RedactorMetadata, error) {
	redactorEntry := RedactorMetadata{}
	redactString, ok := data[slug]
	if !ok {
		return nil, nil, fmt.Errorf("redactor %s not found", slug)
	}

	// unmarshal existing redactor
	err := json.Unmarshal([]byte(redactString), &redactorEntry)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to parse redactor %s", slug)
	}

	redactorEntry.Metadata.Enabled = enabled
	redactorEntry.Metadata.Updated = currentTime

	jsonBytes, err := json.Marshal(redactorEntry)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to marshal redactor %s", slug)
	}

	data[slug] = string(jsonBytes)
	return data, &redactorEntry, nil
}

func setRedactYaml(slug, description string, enabled, newRedact bool, currentTime time.Time, yamlBytes []byte, data map[string]string) (map[string]string, *RedactorMetadata, error) {
	// parse yaml as redactor
	newRedactorSpec, err := parseRedact(yamlBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to parse new redact yaml")
	}

	if data == nil {
		data = map[string]string{}
	}

	redactorEntry := RedactorMetadata{}
	redactString, ok := data[slug]

	if !ok || newRedact {
		// if name is not set in yaml throw error
		// if name is set, create the slug from the name
		if newRedactorSpec.Name == "" {
			return nil, nil, fmt.Errorf("failed to create new redact spec: name can't be empty")
		} else {
			slug = getSlug(newRedactorSpec.Name)
		}

		if _, ok := data[slug]; ok {
			// the target slug already exists - this is an error
			return nil, nil, fmt.Errorf("failed to create new redact spec: name %s - slug %s already exists", newRedactorSpec.Name, slug)
		}

		// create the new redactor
		redactorEntry.Metadata = types.RedactorList{
			Name:    newRedactorSpec.Name,
			Slug:    slug,
			Created: currentTime,
		}
	} else {
		// unmarshal existing redactor, check if name changed
		err = json.Unmarshal([]byte(redactString), &redactorEntry)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to parse redactor %s", slug)
		}

		if slug != getSlug(newRedactorSpec.Name) && newRedactorSpec.Name != "" {
			// changing name

			if _, ok := data[getSlug(newRedactorSpec.Name)]; ok {
				// the target slug already exists - this is an error
				return nil, nil, fmt.Errorf("failed to update redact spec: refusing to change slug from %s to %s as that already exists", slug, getSlug(newRedactorSpec.Name))
			}

			delete(data, slug)
			slug = getSlug(newRedactorSpec.Name)
			redactorEntry.Metadata.Slug = slug
			redactorEntry.Metadata.Name = newRedactorSpec.Name
		}

		if newRedactorSpec.Name == "" {
			return nil, nil, fmt.Errorf("failed to update redact spec: name can't be empty")
		}
	}

	redactorEntry.Metadata.Enabled = enabled
	redactorEntry.Metadata.Description = description
	redactorEntry.Metadata.Updated = currentTime

	redactorEntry.Redact = string(yamlBytes)

	jsonBytes, err := json.Marshal(redactorEntry)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to marshal redactor %s", slug)
	}

	data[slug] = string(jsonBytes)

	return data, &redactorEntry, nil
}

func DeleteRedact(slug string) error {
	configMap, _, err := getRedactConfigmap()
	if err != nil {
		return errors.Wrap(err, "get redactors configmap")
	}

	delete(configMap.Data, slug)

	_, err = writeRedactConfigmap(configMap)
	if err != nil {
		return errors.Wrapf(err, "write configMap with updated redact")
	}
	return nil
}

func getRedactConfigmap() (*v1.ConfigMap, string, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, "failed to get k8s clientset", errors.Wrap(err, "failed to get k8s clientset")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), redactConfigMapName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			// not a not found error, so a real error
			return nil, "failed to get kotsadm-redact configMap", errors.Wrap(err, "failed to get kotsadm-redact configMap")
		} else {
			// not found, so create one and return it
			newMap := v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      redactConfigMapName,
					Namespace: util.PodNamespace,
					Labels: map[string]string{
						"kots.io/kotsadm": "true",
					},
				},
				Data: map[string]string{},
			}
			createdMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), &newMap, metav1.CreateOptions{})
			if err != nil {
				return nil, "failed to create kotsadm-redact configMap", errors.Wrap(err, "failed to create kotsadm-redact configMap")
			}

			return createdMap, "", nil
		}
	}
	return configMap, "", nil
}

// writeRedactConfigmap creates a configmap which contains kotsadm formatted redactors that include some additional metadata (e.g. if a redactor is enabled or not)
func writeRedactConfigmap(configMap *v1.ConfigMap) (*v1.ConfigMap, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	newConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update configmap")
	}
	return newConfigMap, nil
}

func getSlug(name string) string {
	name = slug.Make(name)

	if name == "kotsadm-redact" {
		name = "kotsadm-redact-metadata"
	}
	return name
}

func buildFullRedact(config *v1.ConfigMap) (*troubleshootv1beta2.Redactor, error) {
	full := &troubleshootv1beta2.Redactor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Redactor",
			APIVersion: "troubleshoot.sh/v1beta2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kotsadm-redact",
		},
		Spec: troubleshootv1beta2.RedactorSpec{},
	}

	keys := []string{}
	for k := range config.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := config.Data[k]
		if k == "kotsadm-redact" {
			redactor, err := parseRedact([]byte(v))
			if err == nil && redactor != nil {
				full.Spec.Redactors = append(full.Spec.Redactors, redactor.Spec.Redactors...)
			}
			continue
		}

		redactorEntry := RedactorMetadata{}
		err := json.Unmarshal([]byte(v), &redactorEntry)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse key %s", k)
		}
		if redactorEntry.Metadata.Enabled {
			redactor, err := parseRedact([]byte(redactorEntry.Redact))
			if err != nil {
				return nil, errors.Wrapf(err, "unable to parse redactor %s", k)
			}
			full.Spec.Redactors = append(full.Spec.Redactors, redactor.Spec.Redactors...)
		}
	}
	return full, nil
}

func splitRedactors(spec string) (map[string]string, error) {
	newMap := make(map[string]string, 0)

	redactor, err := parseRedact([]byte(spec))
	if err != nil {
		return nil, errors.Wrap(err, "split redactors")
	}

	for idx, redactorSpec := range redactor.Spec.Redactors {
		if redactorSpec == nil {
			continue
		}

		redactorName := ""
		if redactorSpec.Name != "" {
			redactorName = redactorSpec.Name
		} else {
			redactorName = fmt.Sprintf("redactor-%d", idx)
			redactorSpec.Name = redactorName
		}

		newSpec, err := util.MarshalIndent(2, troubleshootv1beta2.Redactor{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Redactor",
				APIVersion: "troubleshoot.sh/v1beta2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: redactorName,
			},
			Spec: troubleshootv1beta2.RedactorSpec{
				Redactors: []*troubleshootv1beta2.Redact{redactorSpec},
			},
		})

		newRedactor := RedactorMetadata{
			Metadata: types.RedactorList{
				Name:    redactorName,
				Slug:    getSlug(redactorName),
				Created: time.Now(),
				Updated: time.Now(),
				Enabled: true,
			},
			Redact: string(newSpec),
		}

		jsonBytes, err := json.Marshal(newRedactor)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to marshal redactor %s", redactorName)
		}

		newMap[newRedactor.Metadata.Slug] = string(jsonBytes)
	}

	return newMap, nil
}

func parseRedact(spec []byte) (*troubleshootv1beta2.Redactor, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(spec, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "deserialize combined redact spec")
	}
	redactor, ok := obj.(*troubleshootv1beta2.Redactor)
	if ok && redactor != nil {
		return redactor, nil
	}
	return nil, errors.New("not a redactor")
}
