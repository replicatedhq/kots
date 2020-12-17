package deploy

import (
	"context"
	"encoding/base64"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ensureDexThemeConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace string, options Options) error {
	configMap, err := dexThemeConfigMapResource(options)
	if err != nil {
		return err
	}

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMap.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing config map")
		}

		_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create config map")
		}

		return nil
	}

	existingConfigMap = updateDexThemeConfigMap(existingConfigMap, configMap)

	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func dexThemeConfigMapResource(options Options) (*corev1.ConfigMap, error) {
	var styleCSS, logo, favicon []byte
	if options.IdentitySpec.WebConfig != nil && options.IdentitySpec.WebConfig.Theme != nil {
		theme := options.IdentitySpec.WebConfig.Theme

		var err error

		styleCSS, err = base64.StdEncoding.DecodeString(theme.StyleCSSBase64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode style.css")
		}
		logo, err = base64.StdEncoding.DecodeString(theme.LogoBase64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode style.css")
		}
		favicon, err = base64.StdEncoding.DecodeString(theme.FaviconBase64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode style.css")
		}
	}

	data := map[string]string{
		"styles.css": string(styleCSS),
	}
	binaryData := map[string][]byte{
		"logo.png":    logo,
		"favicon.png": favicon,
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   prefixName(options.NamePrefix, "dex-theme"),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(options.NamePrefix, options.AdditionalLabels)),
		},
		Data:       data,
		BinaryData: binaryData,
	}, nil
}

func updateDexThemeConfigMap(existingConfigMap, desiredConfigMap *corev1.ConfigMap) *corev1.ConfigMap {
	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}
	existingConfigMap.Data["styles.css"] = desiredConfigMap.Data["styles.css"]

	if existingConfigMap.BinaryData == nil {
		existingConfigMap.BinaryData = map[string][]byte{}
	}
	existingConfigMap.BinaryData["logo.png"] = desiredConfigMap.BinaryData["logo.png"]
	existingConfigMap.BinaryData["favicon.png"] = desiredConfigMap.BinaryData["favicon.png"]

	return existingConfigMap
}

func deleteDexThemeConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, prefixName(namePrefix, "dex-theme"), metav1.DeleteOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	return err
}
