package auth

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const KotsadmAuthstringSecretName = "kotsadm-authstring"
const KotsadmAuthstringSecretKey = "kotsadm-authstring"

var authSlugCache string

// SetAuthSlugCache sets the auth slug to be used instead of querying kubernetes
// this improves run speed by reducing the number of queries, and also allows testing without a kubernetes cluster available
func SetAuthSlugCache(newval string) {
	authSlugCache = newval
}

// GetOrCreateAuthSlug will check for an authslug secret in the provided namespace
// if one exists, it will return the value from that secret
// if none exists, it will create one and return that value
func GetOrCreateAuthSlug(clientset kubernetes.Interface, namespace string) (string, error) {
	if authSlugCache != "" {
		return authSlugCache, nil
	}

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), KotsadmAuthstringSecretName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			fmt.Printf("\nUnable to authenticate to the Admin Console running in the %s namespace. Ensure you have read access to secrets in this namespace and try again.\n\n", namespace)
			return "", errors.Wrap(err, "failed to check for existing kotsadm authstring secret")
		}

		// secret does not yet exist, so we need to generate a random key and create the secret from that
		newAuthstring := "Kots " + util.GenPassword(32)
		newSecret := v1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      KotsadmAuthstringSecretName,
				Namespace: namespace,
				Labels:    types.GetKotsadmLabels(),
			},
			StringData: map[string]string{KotsadmAuthstringSecretKey: newAuthstring},
		}

		_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), &newSecret, metav1.CreateOptions{})
		if err != nil {
			return "", errors.Wrap(err, "failed to create new kotsadm authstring secret")
		}
		SetAuthSlugCache(newAuthstring)
		return newAuthstring, nil
	}

	SetAuthSlugCache(string(existingSecret.Data[KotsadmAuthstringSecretKey]))
	return string(existingSecret.Data[KotsadmAuthstringSecretKey]), nil
}
