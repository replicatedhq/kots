package kotsadm

import (
	"context"
	"encoding/base64"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/ingress"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

// YAML will return a map containing the YAML needed to run the admin console
func YAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}

	if deployOptions.ApplicationMetadata != nil {
		metadataDocs, err := getApplicationMetadataYAML(deployOptions.ApplicationMetadata, deployOptions.Namespace, deployOptions.UpstreamURI)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get application metadata yaml")
		}
		for n, v := range metadataDocs {
			docs[n] = v
		}
	}

	if deployOptions.IncludeMinio {
		minioDocs, err := getMinioYAML(deployOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get minio yaml")
		}
		for n, v := range minioDocs {
			docs[n] = v
		}
	}

	rqliteDocs, err := getRqliteYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rqlite yaml")
	}
	for n, v := range rqliteDocs {
		docs[n] = v
	}

	// secrets
	secretsDocs, err := getSecretsYAML(&deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secrets yaml")
	}
	for n, v := range secretsDocs {
		docs[n] = v
	}

	// configmaps
	configMapsDocs, err := getConfigMapsYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get configmaps yaml")
	}
	for n, v := range configMapsDocs {
		docs[n] = v
	}

	// kotsadm
	kotsadmDocs, err := getKotsadmYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm yaml")
	}
	for n, v := range kotsadmDocs {
		docs[n] = v
	}

	return docs, nil
}

func Upgrade(clientset *kubernetes.Clientset, upgradeOptions types.UpgradeOptions) error {
	log := logger.NewCLILogger(os.Stdout)

	if err := canUpgrade(upgradeOptions, clientset, log); err != nil {
		log.Error(err)
		return err
	}

	deployOptions, err := ReadDeployOptionsFromCluster(upgradeOptions.Namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to read deploy options")
	}

	// If user has passed in the flag to migrate minio, save this status as part of the install
	// Only works if they have not already chose to opt-in
	if !upgradeOptions.IncludeMinio && deployOptions.IncludeMinioSnapshots {
		deployOptions.IncludeMinioSnapshots = false
	}

	// these options are not stored in cluster (yet)
	deployOptions.Timeout = upgradeOptions.Timeout
	deployOptions.RegistryConfig = upgradeOptions.RegistryConfig
	deployOptions.EnsureRBAC = upgradeOptions.EnsureRBAC
	deployOptions.SimultaneousUploads = upgradeOptions.SimultaneousUploads
	deployOptions.IncludeMinio = upgradeOptions.IncludeMinio
	deployOptions.StrictSecurityContext = upgradeOptions.StrictSecurityContext

	if deployOptions.IncludeMinio {
		deployOptions.MigrateToMinioXl, deployOptions.CurrentMinioImage, err = IsMinioXlMigrationNeeded(clientset, deployOptions.Namespace)
		if err != nil {
			return errors.Wrap(err, "failed to check if minio xl migration is needed")
		}
	}

	// Attempt migrations to fail early.
	if !deployOptions.IncludeMinioSnapshots {
		if err = MigrateExistingMinioFilesystemDeployments(log, deployOptions); err != nil {
			return errors.Wrap(err, "failed to migrate minio filesystem")
		}
	}

	if err := ensureKotsadm(*deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to upgrade admin console")
	}

	if err := removeUnusedKotsadmComponents(*deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to removed unused admin console components")
	}

	return nil
}

func Deploy(deployOptions types.DeployOptions, log *logger.CLILogger) error {
	if deployOptions.AirgapBundle != "" && deployOptions.RegistryConfig.OverrideRegistry != "" {
		pushOptions := imagetypes.PushImagesOptions{
			Registry: registrytypes.RegistryOptions{
				Endpoint:  deployOptions.RegistryConfig.OverrideRegistry,
				Namespace: deployOptions.RegistryConfig.OverrideNamespace,
				Username:  deployOptions.RegistryConfig.Username,
				Password:  deployOptions.RegistryConfig.Password,
			},
			ProgressWriter: deployOptions.ProgressWriter,
		}

		if !deployOptions.DisableImagePush {
			err := image.TagAndPushImagesFromBundle(deployOptions.AirgapBundle, pushOptions)
			if err != nil {
				return errors.Wrap(err, "failed to tag and push app images from path")
			}
		}

		deployOptions.AppImagesPushed = true
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	if deployOptions.AirgapBundle != "" && deployOptions.RegistryConfig.OverrideRegistry == "" {
		log.Info("not pushing airgapped app images as no registry was provided")
	}

	if !deployOptions.ExcludeAdminConsole {
		namespace := &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: deployOptions.Namespace,
			},
		}

		log.ChildActionWithSpinner("Creating namespace")
		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil && !kuberneteserrors.IsAlreadyExists(err) {
			// Can't create namespace, but this might be a role restriction and namespace might already exist.
			_, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return errors.Wrap(err, "failed to verify access to namespace")
			}
		}
		log.FinishChildSpinner()

		if deployOptions.AutoCreateClusterToken == "" {
			deployOptions.AutoCreateClusterToken = uuid.New().String()
		}

		limitRange, err := maybeGetNamespaceLimitRanges(clientset, deployOptions.Namespace)
		if err != nil {
			return errors.Wrap(err, "failed to get limit ranges for namespace")
		}
		deployOptions.LimitRange = limitRange
	}

	if deployOptions.AppImagesPushed {
		airgapMetadata, err := archives.GetFileFromAirgap("airgap.yaml", deployOptions.AirgapBundle)
		if err != nil {
			return errors.Wrap(err, "failed to get airgap.yaml from bundle")
		}
		data := map[string]string{
			"airgap.yaml": base64.StdEncoding.EncodeToString(airgapMetadata),
		}
		if err := ensureConfigMapWithData(deployOptions, clientset, "kotsadm-airgap-meta", data); err != nil {
			return errors.Wrap(err, "failed to create config from airgap.yaml")
		}
		if err := ensureWaitForAirgapConfig(deployOptions, clientset, "kotsadm-airgap-app"); err != nil {
			return errors.Wrap(err, "failed to create config from app.tar.gz")
		}
	}

	if err := ensureKotsadm(deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to deploy admin console")
	}

	return nil
}

func IsAirgap() bool {
	return os.Getenv("DISABLE_OUTBOUND_CONNECTIONS") == "true"
}

func canUpgrade(upgradeOptions types.UpgradeOptions, clientset *kubernetes.Clientset, log *logger.CLILogger) error {
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), upgradeOptions.Namespace, metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err := errors.New("The namespace cannot be found or accessed")
		return err
	}

	if upgradeOptions.ForceUpgradeKurl {
		return nil
	}

	// don't upgrade kurl clusters.  kurl only installs into default namespace.
	if upgradeOptions.Namespace != "" && upgradeOptions.Namespace != "default" {
		return nil
	}

	isKurl, err := kurl.IsKurl(clientset)
	if err != nil {
		return errors.Wrap(err, "failed to check if cluster is kurl")
	}

	if isKurl {
		return errors.New("upgrading kURL clusters is not supported")
	}

	return nil
}

func removeUnusedKotsadmComponents(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.CLILogger) error {
	// if there's a kotsadm web deployment, remove (pre 1.11.0)
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(context.TODO(), "kotsadm-web", metav1.GetOptions{})
	if err == nil {
		if err := clientset.AppsV1().Deployments(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-web", metav1.DeleteOptions{}); err != nil {
			return errors.Wrap(err, "failed to delete kotsadm-web deployment")
		}
	}

	// if there's a service named "kotsadm-api", remove (pre 1.11.0)
	_, err = clientset.CoreV1().Services(deployOptions.Namespace).Get(context.TODO(), "kotsadm-api", metav1.GetOptions{})
	if err == nil {
		if err := clientset.CoreV1().Services(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-api", metav1.DeleteOptions{}); err != nil {
			return errors.Wrap(err, "failed to delete kotsadm-api service")
		}
	}

	// if there are kotsadm-operator objects, remove (pre 1.50.0)
	if err := removeKotsadmOperator(deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to remove kotsadm operator")
	}

	if !deployOptions.IncludeMinio {
		if err := removeKotsadmMinio(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to remove kotsadm minio")
		}
	}

	// if there are kotsadm-postgres objects, remove (pre 1.89.0)
	if err := removeKotsadmPostgres(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to remove kotsadm postgres")
	}

	return nil
}

func removeKotsadmOperator(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.CLILogger) error {
	err := clientset.AppsV1().Deployments(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-operator", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-operator deployment")
	}

	err = clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), "kotsadm-operator-rolebinding", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		// user might not have enough permissions to do so, so don't fail here since it's not critical
		log.Error(errors.Wrap(err, "failed to delete kotsadm-operator-rolebinding clusterrolebinding"))
	}

	err = clientset.RbacV1().RoleBindings(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-operator-rolebinding", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		// user might not have enough permissions to do so, so don't fail here since it's not critical
		log.Error(errors.Wrap(err, "failed to delete kotsadm-operator-rolebinding rolebinding"))
	}

	err = clientset.RbacV1().ClusterRoles().Delete(context.TODO(), "kotsadm-operator-role", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		// user might not have enough permissions to do so, so don't fail here since it's not critical
		log.Error(errors.Wrap(err, "failed to delete kotsadm-operator-role clusterrole"))
	}

	err = clientset.RbacV1().Roles(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-operator-role", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		// user might not have enough permissions to do so, so don't fail here since it's not critical
		log.Error(errors.Wrap(err, "failed to delete kotsadm-operator-role role"))
	}

	// remove roles/rolebindings from the additional namespaces (if applicable)
	if len(deployOptions.ApplicationMetadata) > 0 {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(deployOptions.ApplicationMetadata, nil, nil)
		if err == nil && gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Application" {
			application := obj.(*kotsv1beta1.Application)
			for _, additionalNamespace := range application.Spec.AdditionalNamespaces {
				err := clientset.RbacV1().RoleBindings(additionalNamespace).Delete(context.TODO(), "kotsadm-operator-rolebinding", metav1.DeleteOptions{})
				if err != nil && !kuberneteserrors.IsNotFound(err) {
					// user might not have enough permissions to do so, so don't fail here since it's not critical
					log.Error(errors.Wrapf(err, "failed to delete kotsadm-operator-rolebinding rolebinding in namespace %s", additionalNamespace))
				}
				err = clientset.RbacV1().Roles(additionalNamespace).Delete(context.TODO(), "kotsadm-operator-role", metav1.DeleteOptions{})
				if err != nil && !kuberneteserrors.IsNotFound(err) {
					// user might not have enough permissions to do so, so don't fail here since it's not critical
					log.Error(errors.Wrapf(err, "failed to delete kotsadm-operator-role role in namespace %s", additionalNamespace))
				}
			}
		}
	}

	err = clientset.CoreV1().ServiceAccounts(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-operator", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		// user might not have enough permissions to do so, so don't fail here since it's not critical
		log.Error(errors.Wrap(err, "failed to delete kotsadm-operator serviceaccount"))
	}

	return nil
}

func removeKotsadmMinio(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	// if there's a deployment named "kotsadm", remove (pre 1.47.0)
	// only delete the deployment if minio is not included because that will mean that it's been replaced with a statefulset
	err := clientset.AppsV1().Deployments(deployOptions.Namespace).Delete(context.TODO(), "kotsadm", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm deployment")
	}

	// if there's a service named "kotsadm-minio", remove (pre 1.47.0)
	err = clientset.CoreV1().Services(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-minio", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-minio service")
	}

	// if there's a statefulset named "kotsadm-minio", remove (pre 1.47.0)
	err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-minio", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-minio statefulset")
	}

	// if there's a secret named "kotsadm-minio", remove (pre 1.47.0)
	err = clientset.CoreV1().Secrets(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-minio", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-minio secret")
	}

	// if there's a minio pvc, remove (pre 1.47.0)
	minioPVCSelectorLabels := map[string]string{
		"app": "kotsadm-minio",
	}
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(deployOptions.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(minioPVCSelectorLabels).String(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list kotsadm-minio persistent volume claims")
	}
	for _, pvc := range pvcs.Items {
		err := clientset.CoreV1().PersistentVolumeClaims(deployOptions.Namespace).Delete(context.TODO(), pvc.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to delete kotsadm-minio pvc")
		}
	}

	return nil
}

func removeKotsadmPostgres(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	// if there's a service named "kotsadm-postgres", remove (pre 1.89.0)
	err := clientset.CoreV1().Services(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-postgres", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-postgres service")
	}

	// if there's a statefulset named "kotsadm-postgres", remove (pre 1.89.0)
	err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-postgres", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-postgres statefulset")
	}

	// if there's a configmap named "kotsadm-postgres", remove (pre 1.89.0)
	err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-postgres", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-postgres configmap")
	}

	// if there's a secret named "kotsadm-postgres", remove (pre 1.89.0)
	err = clientset.CoreV1().Secrets(deployOptions.Namespace).Delete(context.TODO(), "kotsadm-postgres", metav1.DeleteOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to delete kotsadm-postgres secret")
	}

	// if there's a postgres pvc, remove (pre 1.89.0)
	postgresPVCSelectorLabels := map[string]string{
		"app": "kotsadm-postgres",
	}
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(deployOptions.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(postgresPVCSelectorLabels).String(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list kotsadm-postgres persistent volume claims")
	}
	for _, pvc := range pvcs.Items {
		err := clientset.CoreV1().PersistentVolumeClaims(deployOptions.Namespace).Delete(context.TODO(), pvc.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to delete kotsadm-postgres pvc")
		}
	}

	return nil
}

func ensureKotsadm(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.CLILogger) error {
	restartKotsadmAPI := false

	ingressConfig := deployOptions.IngressConfig
	identityConfig := deployOptions.IdentityConfig

	if identityConfig.Spec.Enabled {
		if err := identity.ValidateConfig(context.TODO(), deployOptions.Namespace, identityConfig, ingressConfig); err != nil {
			return errors.Wrap(err, "failed to validate identity config")
		}
	}

	// check additional namespaces early in case there are rbac issues we don't
	// leave the cluster in a partially deployed state
	if deployOptions.ApplicationMetadata != nil {
		// If the metadata parses, and if the metadata contains additional namespaces
		// attempt to create
		if err := ensureAdditionalNamespaces(&deployOptions, clientset, log); err != nil {
			return errors.Wrap(err, "failed to ensure additional namespaces")
		}
	}

	if deployOptions.License != nil {
		// if there's a license, we write it as a secret and kotsadm will
		// find it on startup and handle installation
		updated, err := ensureLicenseSecret(&deployOptions, clientset)
		if err != nil {
			return errors.Wrap(err, "failed to ensure license secret")
		}

		if updated {
			restartKotsadmAPI = true
		}
	}

	if deployOptions.ConfigValues != nil {
		// if there's a configvalues file, store it as a secret (they may contain
		// sensitive information) and kotsadm will find it on startup and apply
		// it to the installation
		updated, err := ensureConfigValuesSecret(&deployOptions, clientset)
		if err != nil {
			return errors.Wrap(err, "failed to ensure config values secret")
		}

		if updated {
			restartKotsadmAPI = true
		}
	}

	if deployOptions.ExcludeAdminConsole && deployOptions.EnsureKotsadmConfig {
		if err := ensureKotsadmConfig(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm config")
		}
	}

	if !deployOptions.ExcludeAdminConsole {
		restartKotsadmAPI = false

		if err := ensureKotsadmConfig(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm config")
		}

		g, ctx := errgroup.WithContext(context.TODO())

		g.Go(func() error {
			if deployOptions.IncludeMinio {
				return ensureAndWaitForMinio(ctx, deployOptions, clientset)
			}
			return nil
		})

		g.Go(func() error {
			if err := ensureRqlite(deployOptions, clientset); err != nil {
				return errors.Wrap(err, "failed to ensure rqlite")
			}
			if err := k8sutil.WaitForStatefulSetReady(ctx, clientset, deployOptions.Namespace, "kotsadm-rqlite", deployOptions.Timeout); err != nil {
				return errors.Wrap(err, "failed to wait for rqlite")
			}
			return nil
		})

		log.ChildActionWithSpinner("Waiting for datastore to be ready")
		err := g.Wait()
		log.FinishChildSpinner()
		if err != nil {
			return err
		}

		if err := ensureSecrets(&deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure secrets exist")
		}

		if err := ensureKotsadmComponent(&deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure kotsadm exists")
		}
	}

	if err := ensureApplicationMetadata(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure custom branding")
	}

	if !deployOptions.ExcludeAdminConsole {
		if err := removeNodeAPI(&deployOptions, clientset); err != nil {
			log.Error(errors.Errorf("Failed to remove unused API: %v", err))
		}

		if err := ensureDisasterRecoveryLabels(&deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure disaster recovery labels")
		}
	}

	ctx := context.TODO()

	if ingressConfig.Spec.Enabled {
		log.ChildActionWithSpinner("Enabling ingress for the Admin Console")

		if err := ingress.SetConfig(ctx, deployOptions.Namespace, ingressConfig); err != nil {
			return errors.Wrap(err, "failed to set identity config")
		}

		if err := EnsureIngress(ctx, deployOptions.Namespace, clientset, ingressConfig.Spec); err != nil {
			return errors.Wrap(err, "failed to ensure ingress")
		}

		log.FinishSpinner()
	}

	if identityConfig.Spec.Enabled {
		log.ChildActionWithSpinner("Deploying the Identity Service")

		identityConfig.Spec.DisablePasswordAuth = true

		if identityConfig.Spec.IngressConfig == (kotsv1beta1.IngressConfigSpec{}) {
			identityConfig.Spec.IngressConfig.Enabled = false
		} else {
			identityConfig.Spec.IngressConfig.Enabled = true
		}

		if err := identity.SetConfig(ctx, deployOptions.Namespace, identityConfig); err != nil {
			return errors.Wrap(err, "failed to set identity config")
		}

		proxyEnv := map[string]string{
			"HTTP_PROXY":  deployOptions.HTTPProxyEnvValue,
			"HTTPS_PROXY": deployOptions.HTTPSProxyEnvValue,
			"NO_PROXY":    deployOptions.NoProxyEnvValue,
		}

		isSingleApp := true // TODO (ethan)

		if err := identity.Deploy(ctx, clientset, deployOptions.Namespace, identityConfig, ingressConfig, &deployOptions.RegistryConfig, proxyEnv, isSingleApp); err != nil {
			return errors.Wrap(err, "failed to deploy the identity service")
		}

		log.FinishSpinner()
	}

	if !deployOptions.ExcludeAdminConsole {
		log.ChildActionWithSpinner("Waiting for Admin Console to be ready")
		if deployOptions.IncludeMinio {
			if err := k8sutil.WaitForDeploymentReady(ctx, clientset, deployOptions.Namespace, "kotsadm", deployOptions.Timeout); err != nil {
				return errors.Wrap(err, "failed to wait for web")
			}
		} else {
			if err := k8sutil.WaitForStatefulSetReady(ctx, clientset, deployOptions.Namespace, "kotsadm", deployOptions.Timeout); err != nil {
				return errors.Wrap(err, "failed to wait for web")
			}
		}
		log.FinishSpinner()
	}

	if restartKotsadmAPI {
		log.ChildActionWithSpinner("Waiting for Admin Console to be ready")
		if err := k8sutil.RestartKotsadm(ctx, clientset, deployOptions.Namespace, deployOptions.Timeout); err != nil {
			return errors.Wrap(err, "failed to restart kotsadm")
		}

		if deployOptions.IncludeMinio {
			if err := k8sutil.WaitForDeploymentReady(ctx, clientset, deployOptions.Namespace, "kotsadm", deployOptions.Timeout); err != nil {
				return errors.Wrap(err, "failed to wait for web")
			}
		} else {
			if err := k8sutil.WaitForStatefulSetReady(ctx, clientset, deployOptions.Namespace, "kotsadm", deployOptions.Timeout); err != nil {
				return errors.Wrap(err, "failed to wait for web")
			}
		}
		log.FinishSpinner()
	}

	return nil
}

func ensureDisasterRecoveryLabels(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) error {
	selectorLabels := map[string]string{
		types.KotsadmKey: types.KotsadmLabelValue,
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	// RBAC
	isClusterScoped, err := isKotsadmClusterScoped(deployOptions)
	if err != nil {
		return errors.Wrap(err, "failed to check if kotsadm is cluster scoped")
	}
	if deployOptions.EnsureRBAC && isClusterScoped {
		// cluster roles
		clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list clusterroles")
		}
		for _, clusterRole := range clusterRoles.Items {
			if _, ok := clusterRole.ObjectMeta.Labels[types.BackupLabel]; !ok {
				clusterRole.ObjectMeta.Labels = types.GetKotsadmLabels(clusterRole.ObjectMeta.Labels)

				// remove existing velero exclude label/annotation (if exists)
				delete(clusterRole.ObjectMeta.Labels, types.ExcludeKey)
				delete(clusterRole.ObjectMeta.Annotations, types.ExcludeKey)

				_, err = clientset.RbacV1().ClusterRoles().Update(context.TODO(), &clusterRole, metav1.UpdateOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to update %s clusterrole", clusterRole.ObjectMeta.Name)
				}
			}
		}

		// cluster role bindings
		clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list clusterrolebindings")
		}
		for _, binding := range clusterRoleBinding.Items {
			if _, ok := binding.ObjectMeta.Labels[types.BackupLabel]; !ok {
				binding.ObjectMeta.Labels = types.GetKotsadmLabels(binding.ObjectMeta.Labels)

				// remove existing velero exclude label/annotation (if exists)
				delete(binding.ObjectMeta.Labels, types.ExcludeKey)
				delete(binding.ObjectMeta.Annotations, types.ExcludeKey)

				_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), &binding, metav1.UpdateOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to update %s clusterrolebinding", binding.ObjectMeta.Name)
				}
			}
		}
	} else if deployOptions.EnsureRBAC {
		// roles
		roles, err := clientset.RbacV1().Roles(deployOptions.Namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list roles")
		}
		for _, role := range roles.Items {
			if _, ok := role.ObjectMeta.Labels[types.BackupLabel]; !ok {
				role.ObjectMeta.Labels = types.GetKotsadmLabels(role.ObjectMeta.Labels)

				// remove existing velero exclude label/annotation (if exists)
				delete(role.ObjectMeta.Labels, types.ExcludeKey)
				delete(role.ObjectMeta.Annotations, types.ExcludeKey)

				_, err = clientset.RbacV1().Roles(deployOptions.Namespace).Update(context.TODO(), &role, metav1.UpdateOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to update %s role", role.ObjectMeta.Name)
				}
			}
		}

		// role bindings
		roleBindings, err := clientset.RbacV1().RoleBindings(deployOptions.Namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list rolebindings")
		}
		for _, roleBinding := range roleBindings.Items {
			if _, ok := roleBinding.ObjectMeta.Labels[types.BackupLabel]; !ok {
				roleBinding.ObjectMeta.Labels = types.GetKotsadmLabels(roleBinding.ObjectMeta.Labels)

				// remove existing velero exclude label/annotation (if exists)
				delete(roleBinding.ObjectMeta.Labels, types.ExcludeKey)
				delete(roleBinding.ObjectMeta.Annotations, types.ExcludeKey)

				_, err = clientset.RbacV1().RoleBindings(deployOptions.Namespace).Update(context.TODO(), &roleBinding, metav1.UpdateOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to update %s rolebinding", roleBinding.ObjectMeta.Name)
				}
			}
		}
	}

	// service accounts
	serviceAccounts, err := clientset.CoreV1().ServiceAccounts(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list serviceaccounts")
	}
	for _, serviceAccount := range serviceAccounts.Items {
		if _, ok := serviceAccount.ObjectMeta.Labels[types.BackupLabel]; !ok {
			serviceAccount.ObjectMeta.Labels = types.GetKotsadmLabels(serviceAccount.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(serviceAccount.ObjectMeta.Labels, types.ExcludeKey)
			delete(serviceAccount.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().ServiceAccounts(deployOptions.Namespace).Update(context.TODO(), &serviceAccount, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s serviceaccount in namespace %s", serviceAccount.ObjectMeta.Name, serviceAccount.ObjectMeta.Namespace)
			}
		}
	}

	// PVCs
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list persistent volume claims")
	}
	for _, pvc := range pvcs.Items {
		if _, ok := pvc.ObjectMeta.Labels[types.BackupLabel]; !ok {
			pvc.ObjectMeta.Labels = types.GetKotsadmLabels(pvc.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(pvc.ObjectMeta.Labels, types.ExcludeKey)
			delete(pvc.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().PersistentVolumeClaims(deployOptions.Namespace).Update(context.TODO(), &pvc, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s pvc in namespace %s", pvc.ObjectMeta.Name, pvc.ObjectMeta.Namespace)
			}
		}
	}

	// pods
	pods, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list pods")
	}
	for _, pod := range pods.Items {
		if _, ok := pod.ObjectMeta.Labels[types.BackupLabel]; !ok {
			pod.ObjectMeta.Labels = types.GetKotsadmLabels(pod.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(pod.ObjectMeta.Labels, types.ExcludeKey)
			delete(pod.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().Pods(deployOptions.Namespace).Update(context.TODO(), &pod, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s pod in namespace %s", pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
			}
		}
	}

	// deployments
	deployments, err := clientset.AppsV1().Deployments(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list deployments")
	}
	for _, deployment := range deployments.Items {
		if _, ok := deployment.ObjectMeta.Labels[types.BackupLabel]; !ok {
			// ensure labels
			deployment.ObjectMeta.Labels = types.GetKotsadmLabels(deployment.ObjectMeta.Labels)
			deployment.Spec.Template.ObjectMeta.Labels = types.GetKotsadmLabels(deployment.Spec.Template.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(deployment.ObjectMeta.Labels, types.ExcludeKey)
			delete(deployment.Spec.Template.ObjectMeta.Labels, types.ExcludeKey)
			delete(deployment.ObjectMeta.Annotations, types.ExcludeKey)
			delete(deployment.Spec.Template.ObjectMeta.Annotations, types.ExcludeKey)

			// update deployment
			_, err = clientset.AppsV1().Deployments(deployOptions.Namespace).Update(context.TODO(), &deployment, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s deployment in namespace %s", deployment.ObjectMeta.Name, deployment.ObjectMeta.Namespace)
			}
		}
	}

	// statefulsets
	statefulSets, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list statefulsets")
	}
	for _, statefulSet := range statefulSets.Items {
		if _, ok := statefulSet.ObjectMeta.Labels[types.BackupLabel]; !ok {
			// ensure labels
			statefulSet.ObjectMeta.Labels = types.GetKotsadmLabels(statefulSet.ObjectMeta.Labels)
			statefulSet.Spec.Template.ObjectMeta.Labels = types.GetKotsadmLabels(statefulSet.Spec.Template.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(statefulSet.ObjectMeta.Labels, types.ExcludeKey)
			delete(statefulSet.Spec.Template.ObjectMeta.Labels, types.ExcludeKey)
			delete(statefulSet.ObjectMeta.Annotations, types.ExcludeKey)
			delete(statefulSet.Spec.Template.ObjectMeta.Annotations, types.ExcludeKey)

			// update statefulset
			_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(context.TODO(), &statefulSet, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s statefulSet in namespace %s", statefulSet.ObjectMeta.Name, statefulSet.ObjectMeta.Namespace)
			}
		}
	}

	// secrets
	secrets, err := clientset.CoreV1().Secrets(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list secrets")
	}
	for _, secret := range secrets.Items {
		if _, ok := secret.ObjectMeta.Labels[types.BackupLabel]; !ok {
			secret.ObjectMeta.Labels = types.GetKotsadmLabels(secret.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(secret.ObjectMeta.Labels, types.ExcludeKey)
			delete(secret.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().Secrets(deployOptions.Namespace).Update(context.TODO(), &secret, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s secret in namespace %s", secret.ObjectMeta.Name, secret.ObjectMeta.Namespace)
			}
		}
	}

	// configmaps
	configMaps, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list configmaps")
	}
	for _, configMap := range configMaps.Items {
		if configMap.ObjectMeta.Name == k8sutil.KotsadmIDConfigMapName {
			// don't back up the kotsadm-id configmap so that we don't end up with multiple kotsadm instances with the same id after restoring to other clusters
			continue
		}
		if _, ok := configMap.ObjectMeta.Labels[types.BackupLabel]; !ok {
			configMap.ObjectMeta.Labels = types.GetKotsadmLabels(configMap.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(configMap.ObjectMeta.Labels, types.ExcludeKey)
			delete(configMap.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(context.TODO(), &configMap, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s configmap in namespace %s", configMap.ObjectMeta.Name, configMap.ObjectMeta.Namespace)
			}
		}
	}

	// services
	services, err := clientset.CoreV1().Services(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list services")
	}
	for _, service := range services.Items {
		if _, ok := service.ObjectMeta.Labels[types.BackupLabel]; !ok {
			service.ObjectMeta.Labels = types.GetKotsadmLabels(service.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(service.ObjectMeta.Labels, types.ExcludeKey)
			delete(service.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().Services(deployOptions.Namespace).Update(context.TODO(), &service, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s service in namespace %s", service.ObjectMeta.Name, service.ObjectMeta.Namespace)
			}
		}
	}

	// objects that _did_ not have the kotsadm label set
	// gitops secret
	gitopsSecret, err := clientset.CoreV1().Secrets(deployOptions.Namespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get gitops secret")
	}
	if err == nil {
		if gitopsSecret.ObjectMeta.Labels == nil {
			gitopsSecret.ObjectMeta.Labels = map[string]string{}
		}
		if _, ok := gitopsSecret.ObjectMeta.Labels[types.BackupLabel]; !ok {
			gitopsSecret.ObjectMeta.Labels = types.GetKotsadmLabels(gitopsSecret.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(gitopsSecret.ObjectMeta.Labels, types.ExcludeKey)
			delete(gitopsSecret.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().Secrets(deployOptions.Namespace).Update(context.TODO(), gitopsSecret, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update kotsadm-gitops secret in namespace %s", gitopsSecret.ObjectMeta.Namespace)
			}
		}
	}

	// gitops configmap
	gitopsConfigMap, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(context.TODO(), "kotsadm-gitops", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get gitops configmap")
	}
	if err == nil {
		if gitopsConfigMap.ObjectMeta.Labels == nil {
			gitopsConfigMap.ObjectMeta.Labels = map[string]string{}
		}
		if _, ok := gitopsConfigMap.ObjectMeta.Labels[types.BackupLabel]; !ok {
			gitopsConfigMap.ObjectMeta.Labels = types.GetKotsadmLabels(gitopsConfigMap.ObjectMeta.Labels)

			// remove existing velero exclude label/annotation (if exists)
			delete(gitopsConfigMap.ObjectMeta.Labels, types.ExcludeKey)
			delete(gitopsConfigMap.ObjectMeta.Annotations, types.ExcludeKey)

			_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(context.TODO(), gitopsConfigMap, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update kotsadm-gitops configmap in namespace %s", gitopsConfigMap.ObjectMeta.Namespace)
			}
		}
	}

	return nil
}

func ReadDeployOptionsFromCluster(namespace string, clientset *kubernetes.Clientset) (*types.DeployOptions, error) {
	deployOptions := types.DeployOptions{
		Namespace:      namespace,
		ServiceType:    "ClusterIP",
		IsOpenShift:    k8sutil.IsOpenShift(clientset),
		IsGKEAutopilot: k8sutil.IsGKEAutopilot(clientset),
	}

	kotsInstallID, err := getKotsInstallID(deployOptions.Namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kots install id")
	}
	deployOptions.InstallID = kotsInstallID

	// Shared password, we can't read the original, but we can check if there's a bcrypted value
	// the caller should not recreate if there is a password bcrypt on the return value
	sharedPasswordSecret, err := getSharedPasswordSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get shared password secret")
	}
	if sharedPasswordSecret != nil {
		data, ok := sharedPasswordSecret.Data["passwordBcrypt"]
		if ok {
			deployOptions.SharedPasswordBcrypt = string(data)
		}
	}
	if deployOptions.SharedPasswordBcrypt == "" {
		sharedPassword, err := util.PromptForNewPassword()
		if err != nil {
			return nil, errors.Wrap(err, "failed to prompt for shared password")
		}

		deployOptions.SharedPassword = sharedPassword
	}

	if deployOptions.IncludeMinio {
		// s3 secret, get from cluster or create new random values
		s3Secret, err := getS3Secret(namespace, clientset)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get s3 secret")
		}
		if s3Secret != nil {
			accessKey, ok := s3Secret.Data["accesskey"]
			if ok {
				deployOptions.S3AccessKey = string(accessKey)
			}

			secretyKey, ok := s3Secret.Data["secretkey"]
			if ok {
				deployOptions.S3SecretKey = string(secretyKey)
			}
		}
		if deployOptions.S3AccessKey == "" {
			deployOptions.S3AccessKey = uuid.New().String()
		}
		if deployOptions.S3SecretKey == "" {
			deployOptions.S3SecretKey = uuid.New().String()
		}
	}

	// jwt key, get or create new value
	jwtSecret, err := getJWTSessionSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get jwt secret")
	}
	if jwtSecret != nil {
		sessionKey, ok := jwtSecret.Data["key"]
		if ok {
			deployOptions.JWT = string(sessionKey)
		}
	}

	// rqlite password, read from the secret or create new password
	rqliteSecret, err := getRqliteSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rqlite secret")
	}
	if rqliteSecret != nil {
		password, ok := rqliteSecret.Data["password"]
		if ok {
			deployOptions.RqlitePassword = string(password)
		}
	}

	// API encryption key, read from the secret or create new password
	encyptionSecret, err := getAPIEncryptionSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get api encryption secret")
	}
	if encyptionSecret != nil {
		key, ok := encyptionSecret.Data["encryptionKey"]
		if ok {
			deployOptions.APIEncryptionKey = string(key)
		}
	}

	// AutoCreateClusterToken
	autocreateClusterToken, err := getAPIAutoCreateClusterToken(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get auto create cluster token")
	}
	if autocreateClusterToken != "" {
		deployOptions.AutoCreateClusterToken = autocreateClusterToken
	}

	metadataConfig, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(context.TODO(), "kotsadm-application-metadata", metav1.GetOptions{})
	if err == nil {
		deployOptions.ApplicationMetadata = []byte(metadataConfig.Data["application.yaml"])
	} else if !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get app metadata from configmap")
	}

	// Get minio snapshot migration status v1.50.0
	kostadmConfig, err := clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Get(context.TODO(), types.KotsadmConfigMap, metav1.GetOptions{})
	if err == nil {
		var includeMinioSnapshots bool
		includeMinioSnapshotStr, ok := kostadmConfig.Data["minio-enabled-snapshots"]

		if ok {
			includeMinioSnapshots, err = strconv.ParseBool(includeMinioSnapshotStr)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse minio-enabled-snapshots")
			}
		} else {
			includeMinioSnapshots = true
		}

		deployOptions.IncludeMinioSnapshots = includeMinioSnapshots
	} else if kuberneteserrors.IsNotFound(err) {
		deployOptions.IncludeMinioSnapshots = true
	} else {
		return nil, errors.Wrap(err, "failed to get kotsadm config from configmap")
	}

	identityConfig, err := identity.GetConfig(context.TODO(), deployOptions.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get identity config")
	}
	if identityConfig != nil {
		deployOptions.IdentityConfig = *identityConfig
	}

	ingressConfig, err := ingress.GetConfig(context.TODO(), deployOptions.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ingress config")
	}
	if ingressConfig != nil {
		deployOptions.IngressConfig = *ingressConfig
	}

	return &deployOptions, nil
}

func GetRegistryConfigFromCluster(namespace string, clientset kubernetes.Interface) (types.RegistryConfig, error) {
	registryConfig := types.RegistryConfig{}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), types.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return registryConfig, nil
		}
		return registryConfig, errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	// this can be set even if there is no registry endpoint
	if configMap.Data["registry-is-read-only"] == "true" {
		registryConfig.IsReadOnly = true
	}

	endpoint := configMap.Data["kotsadm-registry"]
	if endpoint == "" {
		return registryConfig, nil
	}

	parts := strings.Split(endpoint, "/")
	registryConfig.OverrideRegistry = parts[0]
	if len(parts) > 1 {
		registryConfig.OverrideNamespace = path.Join(parts[1:]...)
	}

	imagePullSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), types.PrivateKotsadmRegistrySecret, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return registryConfig, nil
		}
		return registryConfig, errors.Wrap(err, "failed to get existing private kotsadm registry secret")
	}

	dockerConfigJson := imagePullSecret.Data[".dockerconfigjson"]
	if len(dockerConfigJson) == 0 {
		return registryConfig, nil
	}

	creds, err := registry.GetCredentialsForRegistryFromConfigJSON(dockerConfigJson, registryConfig.OverrideRegistry)
	if err != nil {
		return registryConfig, errors.Wrap(err, "failed to parse dockerconfigjson")
	}

	registryConfig.Username = creds.Username
	registryConfig.Password = creds.Password
	return registryConfig, nil
}
