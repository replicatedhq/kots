package redact

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8s"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/render/helper"
	"github.com/replicatedhq/kots/pkg/store"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func GetAppRedactSpecConfigMapName(appSlug string) string {
	return fmt.Sprintf("kotsadm-%s-redact-spec", appSlug)
}

func GetAppRedactSpecURI(appSlug string) string {
	return fmt.Sprintf("configmap/%s/%s/%s", os.Getenv("POD_NAMESPACE"), GetAppRedactSpecConfigMapName(appSlug), redactSpecDataKey)
}

// WriteAppRedactSpecConfigMap creates a configmap that contains the redaction yaml spec included in the application release
func WriteAppRedactSpecConfigMap(appID string, sequence int64, kotsKinds *kotsutil.KotsKinds) error {
	builtRedactor := kotsKinds.Redactor.DeepCopy()
	if builtRedactor == nil {
		builtRedactor = &troubleshootv1beta2.Redactor{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Redactor",
				APIVersion: "troubleshoot.sh/v1beta2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-redactor",
			},
		}
	}

	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(builtRedactor, &b); err != nil {
		return errors.Wrap(err, "failed to encode redactor")
	}
	templatedSpec := b.Bytes()

	rs, err := helper.RenderAppFile(app, &sequence, templatedSpec, kotsKinds)
	if err != nil {
		return errors.Wrap(err, "failed render redactor spec")
	}
	renderedSpec := string(rs)

	clientset, err := k8s.Clientset()
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	configMapName := GetAppRedactSpecConfigMapName(app.Slug)

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read redactor configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		configMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: os.Getenv("POD_NAMESPACE"),
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string]string{
				redactSpecDataKey: renderedSpec,
			},
		}

		_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create redactor configmap")
		}

		return nil
	}

	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}
	existingConfigMap.Data[redactSpecDataKey] = renderedSpec
	existingConfigMap.ObjectMeta.Labels = kotsadmtypes.GetKotsadmLabels()

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update redactor configmap")
	}

	return nil
}
