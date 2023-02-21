package migrate

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/store"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func RunMigrations(ctx context.Context, namespace string) error {
	if err := migrateKotsadmDexFromPostgresToInCluster(ctx, namespace); err != nil {
		return fmt.Errorf("failed to migrate kotsadm dex from postgres to in cluster: %w", err)
	}

	return nil
}

func migrateKotsadmDexFromPostgresToInCluster(ctx context.Context, namespace string) error {
	identityConfig, err := identity.GetConfig(ctx, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get identity config")
	}
	if identityConfig == nil || !identityConfig.Spec.Enabled {
		return nil
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	postgresSecretExists, err := postgresSecretExists(ctx, clientset, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to check if postgres secret exists")
	}
	if !postgresSecretExists {
		return nil
	}

	ingressConfig, err := ingress.GetConfig(ctx, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get ingress config")
	}

	registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to get registry config from cluster")
	}

	apps, err := store.GetStore().ListInstalledAppSlugs()
	if err != nil {
		return errors.Wrap(err, "failed to list installed apps")
	}
	applyAppBranding := len(apps) == 1

	proxyEnv := map[string]string{
		"HTTP_PROXY":  os.Getenv("HTTP_PROXY"),
		"HTTPS_PROXY": os.Getenv("HTTPS_PROXY"),
		"NO_PROXY":    os.Getenv("NO_PROXY"),
	}
	if err := identity.Deploy(ctx, clientset, namespace, *identityConfig, *ingressConfig, &registryConfig, proxyEnv, applyAppBranding); err != nil {
		return errors.Wrap(err, "failed to deploy identity")
	}

	if err := deletePostgresSecret(ctx, clientset, namespace); err != nil {
		return errors.Wrap(err, "failed to delete postgres secret")
	}

	return nil
}

func postgresSecretExists(ctx context.Context, clientset kubernetes.Interface, namespace string) (bool, error) {
	_, err := clientset.CoreV1().Secrets(namespace).Get(ctx, "kotsadm-dex-postgres", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return false, errors.Wrap(err, "failed to get postgres secret")
	}
	return err == nil, nil
}

func deletePostgresSecret(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(ctx, "kotsadm-dex-postgres", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete postgres secret")
	}
	return nil
}
