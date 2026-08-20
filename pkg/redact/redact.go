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
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/redact/types"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	k8syaml "sigs.k8s.io/yaml"
)

func init() {
	troubleshootscheme.AddToScheme(scheme.Scheme)
}

type RedactorMetadata struct {
	Metadata types.RedactorList `json:"metadata"`

	Redact string `json:"redact"`
}

const (
	redactSecretName  = "kotsadm-redact"
	redactSpecName    = "kotsadm-redact-spec"
	redactSpecDataKey = "redact-spec"
)

func GetKotsadmRedactSpecURI() string {
	return fmt.Sprintf("secret/%s/%s/%s", util.PodNamespace, redactSpecName, redactSpecDataKey)
}

// deleteLegacyRedactSpecConfigMap deletes a legacy ConfigMap that was used to store a
// rendered redactor spec before redactor specs were moved to Secrets.
func deleteLegacyRedactSpecConfigMap(clientset kubernetes.Interface, name string) {
	err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		logger.Errorf("failed to delete legacy redactor spec configmap %s: %v", name, err)
	}
}

// GenerateKotsadmRedactSpec creates a secret that contains the admin console custom redaction yaml spec
// generated from "kotsadm-redact" secret for collecting support bundles. contains the full redact spec type that is supported by troubleshoot.
func GenerateKotsadmRedactSpec(clientset kubernetes.Interface) error {
	spec, _, err := GetRedactSpec()
	if err != nil {
		return errors.Wrap(err, "failed to get redact spec")
	}

	existingSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), redactSpecName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read redact spec secret")
	} else if kuberneteserrors.IsNotFound(err) {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      redactSpecName,
				Namespace: util.PodNamespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string][]byte{
				redactSpecDataKey: []byte(spec),
			},
		}

		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create redactor spec secret")
		}

		deleteLegacyRedactSpecConfigMap(clientset, redactSpecName)
		return nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}
	existingSecret.Data[redactSpecDataKey] = []byte(spec)
	existingSecret.ObjectMeta.Labels = kotsadmtypes.GetKotsadmLabels()

	_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update redactor spec secret")
	}

	deleteLegacyRedactSpecConfigMap(clientset, redactSpecName)
	return nil
}

// GetRedactSpec returns the redaction yaml spec, a pretty error string, and the underlying error
func GetRedactSpec() (string, string, error) {
	secret, errstr, err := getRedactSecret()
	if err != nil || secret == nil {
		return "", errstr, errors.Wrap(err, "get redactors secret")
	}

	return getRedactSpec(secret)
}

func getRedactSpec(secret *corev1.Secret) (string, string, error) {
	redactObj, err := buildFullRedact(secret)
	if err != nil {
		return "", "failed to build full redact yaml", err
	}

	b, err := k8syaml.Marshal(redactObj)
	if err != nil {
		return "", "failed to marshal full redact yaml", err
	}

	// NOTE(ethan): I am not sure why this is necessary but I'm not going to change it
	b, err = kotsutil.FixUpYAML(b)
	if err != nil {
		return "", "failed to fix up full redact yaml", err
	}

	return string(b), "", nil
}

func GetRedact() (*troubleshootv1beta2.Redactor, error) {
	secret, _, err := getRedactSecret()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors secret")
	}
	if secret == nil {
		return nil, nil
	}

	return buildFullRedact(secret)
}

func GetRedactInfo() ([]types.RedactorList, error) {
	secret, _, err := getRedactSecret()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors secret")
	}
	if secret == nil {
		return nil, nil
	}

	if combinedYaml, ok := secret.Data["kotsadm-redact"]; ok {
		// this is the key used for the combined redact list, so run the migration
		newMap, err := splitRedactors(string(combinedYaml))
		if err != nil {
			return nil, errors.Wrap(err, "failed to split combined redactors")
		}
		secret.Data = stringMapToSecretData(newMap)

		// now that the redactors have been split, save the secret
		secret, err = writeRedactSecret(secret)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update secret")
		}
	}

	list := []types.RedactorList{}

	for k, v := range secret.Data {
		redactorEntry := RedactorMetadata{}
		err = json.Unmarshal(v, &redactorEntry)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse key %s", k)
		}
		list = append(list, redactorEntry.Metadata)
	}
	return list, nil
}

func GetRedactBySlug(slug string) (*RedactorMetadata, error) {
	secret, _, err := getRedactSecret()
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, errors.Wrap(err, "get redactors secret")
	}

	redactBytes, ok := secret.Data[slug]
	if !ok {
		return nil, fmt.Errorf("redactor %s not found", slug)
	}

	redactorEntry := RedactorMetadata{}
	err = json.Unmarshal(redactBytes, &redactorEntry)
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

	secret, errMsg, err := getRedactSecret()
	if err != nil {
		return errMsg, errors.Wrap(err, "get redactors secret")
	}

	newMap, err := splitRedactors(spec)
	if err != nil {
		return "failed to split redactors", errors.Wrap(err, "failed to split redactors")
	}

	secret.Data = stringMapToSecretData(newMap)
	_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return "failed to update kotsadm-redact secret", errors.Wrap(err, "failed to update kotsadm-redact secret")
	}
	return "", nil
}

// updates/creates an individual redact with the provided metadata and yaml
func SetRedactYaml(slug, description string, enabled, newRedact bool, yamlBytes []byte) (*RedactorMetadata, error) {
	secret, _, err := getRedactSecret()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors secret")
	}

	newData, redactorEntry, err := setRedactYaml(slug, description, enabled, newRedact, time.Now(), yamlBytes, secretDataToStringMap(secret.Data))
	if err != nil {
		return nil, err
	}

	secret.Data = stringMapToSecretData(newData)

	_, err = writeRedactSecret(secret)
	if err != nil {
		return nil, errors.Wrapf(err, "write secret with updated redact")
	}
	return redactorEntry, nil
}

// sets whether an individual redactor is enabled
func SetRedactEnabled(slug string, enabled bool) (*RedactorMetadata, error) {
	secret, _, err := getRedactSecret()
	if err != nil {
		return nil, errors.Wrap(err, "get redactors secret")
	}

	newData, redactorEntry, err := setRedactEnabled(slug, enabled, time.Now(), secretDataToStringMap(secret.Data))
	if err != nil {
		return nil, err
	}

	secret.Data = stringMapToSecretData(newData)

	_, err = writeRedactSecret(secret)
	if err != nil {
		return nil, errors.Wrapf(err, "write secret with updated redact")
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
		return nil, nil, errors.Wrapf(err, "unable to marshal redactor metadata %s", slug)
	}

	data[slug] = string(jsonBytes)

	return data, &redactorEntry, nil
}

func DeleteRedact(slug string) error {
	secret, _, err := getRedactSecret()
	if err != nil {
		return errors.Wrap(err, "get redactors secret")
	}

	delete(secret.Data, slug)

	_, err = writeRedactSecret(secret)
	if err != nil {
		return errors.Wrapf(err, "write secret with updated redact")
	}
	return nil
}

// MigrateRedactConfigMap creates the kotsadm-redact Secret from the legacy
// kotsadm-redact ConfigMap if it exists, then deletes the ConfigMap.
func MigrateRedactConfigMap(clientset kubernetes.Interface) error {
	configMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), redactSecretName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get kotsadm-redact configmap")
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), redactSecretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get kotsadm-redact secret")
	}

	if kuberneteserrors.IsNotFound(err) {
		data := map[string][]byte{}
		for k, v := range configMap.Data {
			data[k] = []byte(v)
		}
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      redactSecretName,
				Namespace: util.PodNamespace,
				Labels:    configMap.Labels,
			},
			Data: data,
		}
		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create kotsadm-redact secret")
		}
	} else {
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		for k, v := range configMap.Data {
			if _, ok := secret.Data[k]; !ok {
				secret.Data[k] = []byte(v)
			}
		}
		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update kotsadm-redact secret")
		}
	}

	if err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Delete(context.TODO(), redactSecretName, metav1.DeleteOptions{}); err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to delete kotsadm-redact configmap")
		}
	}
	return nil
}

func getRedactSecret() (*corev1.Secret, string, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, "failed to get k8s clientset", errors.Wrap(err, "failed to get k8s clientset")
	}

	// Migrate the legacy ConfigMap if it exists before reading the Secret.
	if err := MigrateRedactConfigMap(clientset); err != nil {
		return nil, "failed to migrate kotsadm-redact configmap", err
	}

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), redactSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			// not a not found error, so a real error
			return nil, "failed to get kotsadm-redact secret", errors.Wrap(err, "failed to get kotsadm-redact secret")
		}
		// not found, so create one and return it
		newSecret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      redactSecretName,
				Namespace: util.PodNamespace,
				Labels: map[string]string{
					"kots.io/kotsadm": "true",
				},
			},
			Data: map[string][]byte{},
		}
		createdSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), &newSecret, metav1.CreateOptions{})
		if err != nil {
			return nil, "failed to create kotsadm-redact secret", errors.Wrap(err, "failed to create kotsadm-redact secret")
		}

		return createdSecret, "", nil
	}
	return secret, "", nil
}

// writeRedactSecret creates/updates a secret which contains kotsadm formatted redactors that include some additional metadata (e.g. if a redactor is enabled or not)
func writeRedactSecret(secret *corev1.Secret) (*corev1.Secret, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	newSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update secret")
	}
	return newSecret, nil
}

func secretDataToStringMap(data map[string][]byte) map[string]string {
	if data == nil {
		return nil
	}
	result := map[string]string{}
	for k, v := range data {
		result[k] = string(v)
	}
	return result
}

func stringMapToSecretData(data map[string]string) map[string][]byte {
	if data == nil {
		return nil
	}
	result := map[string][]byte{}
	for k, v := range data {
		result[k] = []byte(v)
	}
	return result
}

func getSlug(name string) string {
	name = slug.Make(name)

	if name == "kotsadm-redact" {
		name = "kotsadm-redact-metadata"
	}
	return name
}

func buildFullRedact(secret *corev1.Secret) (*troubleshootv1beta2.Redactor, error) {
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
	for k := range secret.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := string(secret.Data[k])
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

		newSpec := troubleshootv1beta2.Redactor{
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
		}

		b, err := k8syaml.Marshal(newSpec)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to marshal redactor %s", redactorName)
		}

		// NOTE(ethan): I am not sure why this is necessary but I'm not going to change it
		b, err = kotsutil.FixUpYAML(b)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to fix up redactor %s", redactorName)
		}

		newRedactor := RedactorMetadata{
			Metadata: types.RedactorList{
				Name:    redactorName,
				Slug:    getSlug(redactorName),
				Created: time.Now(),
				Updated: time.Now(),
				Enabled: true,
			},
			Redact: string(b),
		}

		jsonBytes, err := json.Marshal(newRedactor)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to marshal redactor metadata %s", redactorName)
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
