package kotsadm

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

// YAML will return a map containing the YAML needed to run the admin console
func YAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}

	if deployOptions.ApplicationMetadata != nil {
		metadataDocs, err := getApplicationMetadataYAML(deployOptions.ApplicationMetadata, deployOptions.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get application metadata yaml")
		}
		for n, v := range metadataDocs {
			docs[n] = v
		}
	}

	minioDocs, err := getMinioYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio yaml")
	}
	for n, v := range minioDocs {
		docs[n] = v
	}

	postgresDocs, err := getPostgresYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres yaml")
	}
	for n, v := range postgresDocs {
		docs[n] = v
	}

	migrationDocs, err := getMigrationsYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get migrations yaml")
	}
	for n, v := range migrationDocs {
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

	// kotsadm
	kotsadmDocs, err := getKotsadmYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm yaml")
	}
	for n, v := range kotsadmDocs {
		docs[n] = v
	}

	// operator
	operatorDocs, err := getOperatorYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get operator yaml")
	}
	for n, v := range operatorDocs {
		docs[n] = v
	}

	return docs, nil
}

func Upgrade(upgradeOptions types.UpgradeOptions) error {
	clientset, err := k8sutil.GetClientset(upgradeOptions.KubernetesConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	log := logger.NewLogger()

	if err := canUpgrade(upgradeOptions, clientset, log); err != nil {
		log.Error(err)
		return err
	}

	deployOptions, err := readDeployOptionsFromCluster(upgradeOptions.Namespace, upgradeOptions.KubernetesConfigFlags, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to read deploy options")
	}

	// these options are not stored in cluster (yet)
	deployOptions.Timeout = upgradeOptions.Timeout
	deployOptions.KotsadmOptions = upgradeOptions.KotsadmOptions

	if err := ensureKotsadm(*deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to upgrade admin console")
	}

	if err := removeUnusedKotsadmComponents(*deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to removed unused admin console components")
	}

	return nil
}

func Deploy(deployOptions types.DeployOptions) error {

	airgapPath := ""
	var images []kustomizetypes.Image

	if deployOptions.AirgapArchive != "" && deployOptions.KotsadmOptions.OverrideRegistry != "" {
		airgapRootDir, err := ioutil.TempDir("", "kotsadm-airgap")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(airgapRootDir)

		err = ExtractAirgapImages(deployOptions.AirgapArchive, airgapRootDir, deployOptions.ProgressWriter)
		if err != nil {
			return errors.Wrap(err, "failed to extract images")
		}

		pushOptions := types.PushImagesOptions{
			Registry: registry.RegistryOptions{
				Endpoint:  deployOptions.KotsadmOptions.OverrideRegistry,
				Namespace: deployOptions.KotsadmOptions.OverrideNamespace,
				Username:  deployOptions.KotsadmOptions.Username,
				Password:  deployOptions.KotsadmOptions.Password,
			},
			ProgressWriter: deployOptions.ProgressWriter,
		}

		imagesRootDir := filepath.Join(airgapRootDir, "images")
		images, err = TagAndPushAppImages(imagesRootDir, pushOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list image formats")
		}

		airgapPath = airgapRootDir
	}

	clientset, err := k8sutil.GetClientset(deployOptions.KubernetesConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	log := logger.NewLogger()
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

	deployOptions.IsOpenShift = isOpenshift(clientset)

	if airgapPath != "" {
		deployOptions.AppImagesPushed = true

		b, err := json.Marshal(images)
		if err != nil {
			return errors.Wrap(err, "failed to marshal images data")
		}
		err = ioutil.WriteFile(filepath.Join(airgapPath, "images.json"), b, 0644)
		if err != nil {
			return errors.Wrap(err, "failed to write images data")
		}

		if err := ensureConfigFromFile(deployOptions, clientset, "kotsadm-airgap-meta", filepath.Join(airgapPath, "airgap.yaml")); err != nil {
			return errors.Wrap(err, "failed to create config from airgap.yaml")
		}
		if err := ensureConfigFromFile(deployOptions, clientset, "kotsadm-airgap-app", filepath.Join(airgapPath, "app.tar.gz")); err != nil {
			return errors.Wrap(err, "failed to create config from app.tar.gz")
		}
		if err := ensureConfigFromFile(deployOptions, clientset, "kotsadm-airgap-images", filepath.Join(airgapPath, "images.json")); err != nil {
			return errors.Wrap(err, "failed to create config from images.json")
		}
	}

	if err := ensureStorage(deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to deplioyt backing storage")
	}

	if err := ensureKotsadm(deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to deploy admin console")
	}

	return nil
}

func CreateRestoreJob(options *types.RestoreJobOptions) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to get client set")
	}

	namespace := os.Getenv("POD_NAMESPACE")
	isOpenShift := isOpenshift(clientset)

	kotsadmOptions, err := GetKotsadmOptionsFromCluster(namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to get kotsadm options from cluster")
	}

	job := restoreJob(options.BackupName, namespace, isOpenShift, kotsadmOptions)
	_, err = clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create restore job")
	}

	return nil
}

func IsKurl(k8sConfigFlags *genericclioptions.ConfigFlags) (bool, error) {
	clientset, err := k8sutil.GetClientset(k8sConfigFlags)
	if err != nil {
		return false, errors.Wrap(err, "failed to get clientset")
	}

	log := logger.NewLogger()

	return isKurl(clientset, log), nil
}

func canUpgrade(upgradeOptions types.UpgradeOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
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

	if isKurl(clientset, log) {
		return errors.New("upgrading kURL clusters is not supported")
	}

	return nil
}

func isKurl(clientset *kubernetes.Clientset, log *logger.Logger) bool {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(errors.Wrap(err, "failed to list nodes"))
		return false
	}

	for _, node := range nodes.Items {
		for k, v := range node.Labels {
			if k == "kurl.sh/cluster" && v == "true" {
				return true
			}
		}
	}

	return false
}

func removeUnusedKotsadmComponents(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
	// if there's a kotsadm web deployment, rmove (pre 1.11.0)
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

	return nil
}

func ensureStorage(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
	if deployOptions.ExcludeAdminConsole {
		return nil
	}

	if deployOptions.IncludeDockerDistribution {
		if err := ensureDistribution(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure docker distribution")
		}
	} else if deployOptions.IncludeMinio {
		// note that this is an else if.  if docker distribution _replaces_ minio
		// in a kots install
		if err := ensureMinio(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure minio")
		}
	}

	return nil
}

func ensureKotsadm(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
	restartKotsadmAPI := false

	existingDeployment, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get existing deployment")
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

		if err := ensurePostgres(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure postgres")
		}

		if err := runSchemaHeroMigrations(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to run database migrations")
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
		if err := ensureOperator(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure operator")
		}

		if err := removeNodeAPI(&deployOptions, clientset); err != nil {
			log.Error(errors.Errorf("Failed to remove unused API: %v", err))
		}

		if err := ensureDisasterRecoveryLabels(&deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure disaster recovery labels")
		}

		log.ChildActionWithSpinner("Waiting for Admin Console to be ready")
		if err := waitForKotsadm(&deployOptions, existingDeployment, clientset); err != nil {
			return errors.Wrap(err, "failed to wait for web")
		}
		log.FinishSpinner()
	}

	if restartKotsadmAPI {
		log.ChildActionWithSpinner("Waiting for Admin Console to be ready")
		if err := restartKotsadm(&deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to wait for web")
		}

		if err := waitForKotsadm(&deployOptions, existingDeployment, clientset); err != nil {
			return errors.Wrap(err, "failed to wait for web")
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
	isClusterScoped, err := isKotsadmClusterScoped(deployOptions.ApplicationMetadata) // TODO: does ApplicationMetadata exist?
	if err != nil {
		return errors.Wrap(err, "failed to check if kotsadm is cluster scoped")
	}
	if isClusterScoped {
		// cluster roles
		clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list clusterroles")
		}
		for _, clusterRole := range clusterRoles.Items {
			if _, ok := clusterRole.ObjectMeta.Labels[types.BackupLabel]; !ok {
				clusterRole.ObjectMeta.Labels = types.GetKotsadmLabels(clusterRole.ObjectMeta.Labels)
				delete(clusterRole.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
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
				delete(binding.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
				_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), &binding, metav1.UpdateOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to update %s clusterrolebinding", binding.ObjectMeta.Name)
				}
			}
		}
	} else {
		// roles
		roles, err := clientset.RbacV1().Roles(deployOptions.Namespace).List(context.TODO(), listOptions)
		if err != nil {
			return errors.Wrap(err, "failed to list roles")
		}
		for _, role := range roles.Items {
			if _, ok := role.ObjectMeta.Labels[types.BackupLabel]; !ok {
				role.ObjectMeta.Labels = types.GetKotsadmLabels(role.ObjectMeta.Labels)
				delete(role.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
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
				delete(roleBinding.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
				_, err = clientset.RbacV1().RoleBindings(deployOptions.Namespace).Update(context.TODO(), &roleBinding, metav1.UpdateOptions{})
				if err != nil {
					return errors.Wrapf(err, "failed to update %s rolebinding", roleBinding.ObjectMeta.Name)
				}
			}
		}
	}

	// service accounts
	serviceAccount, err := clientset.CoreV1().ServiceAccounts(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list serviceaccounts")
	}
	for _, serviceAccount := range serviceAccount.Items {
		if _, ok := serviceAccount.ObjectMeta.Labels[types.BackupLabel]; !ok {
			serviceAccount.ObjectMeta.Labels = types.GetKotsadmLabels(serviceAccount.ObjectMeta.Labels)
			delete(serviceAccount.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
			_, err = clientset.CoreV1().ServiceAccounts(deployOptions.Namespace).Update(context.TODO(), &serviceAccount, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s serviceaccount in namespace %s", serviceAccount.ObjectMeta.Name, serviceAccount.ObjectMeta.Namespace)
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

			// remove existing velero exclude label (if exists)
			delete(deployment.ObjectMeta.Labels, types.ExcludeLabel)
			delete(deployment.Spec.Template.ObjectMeta.Labels, types.ExcludeLabel)

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

			// remove existing velero exclude label (if exists)
			delete(statefulSet.ObjectMeta.Labels, types.ExcludeLabel)
			delete(statefulSet.Spec.Template.ObjectMeta.Labels, types.ExcludeLabel)

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
			delete(secret.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
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
		if _, ok := configMap.ObjectMeta.Labels[types.BackupLabel]; !ok {
			configMap.ObjectMeta.Labels = types.GetKotsadmLabels(configMap.ObjectMeta.Labels)
			delete(configMap.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
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
			delete(service.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
			_, err = clientset.CoreV1().Services(deployOptions.Namespace).Update(context.TODO(), &service, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s service in namespace %s", service.ObjectMeta.Name, service.ObjectMeta.Namespace)
			}
		}
	}

	// jobs
	jobs, err := clientset.BatchV1().Jobs(deployOptions.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list jobs")
	}
	for _, job := range jobs.Items {
		if _, ok := job.ObjectMeta.Labels[types.BackupLabel]; !ok {
			job.ObjectMeta.Labels = types.GetKotsadmLabels(job.ObjectMeta.Labels)

			// ensure labels
			job.ObjectMeta.Labels = types.GetKotsadmLabels(job.ObjectMeta.Labels)
			job.Spec.Template.ObjectMeta.Labels = types.GetKotsadmLabels(job.Spec.Template.ObjectMeta.Labels)

			// remove existing velero exclude label (if exists)
			delete(job.ObjectMeta.Labels, types.ExcludeLabel)
			delete(job.Spec.Template.ObjectMeta.Labels, types.ExcludeLabel)

			_, err = clientset.BatchV1().Jobs(deployOptions.Namespace).Update(context.TODO(), &job, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update %s job in namespace %s", job.ObjectMeta.Name, job.ObjectMeta.Namespace)
			}
		}
	}

	// objects that do not have the kotsadm label set
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
			delete(gitopsSecret.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
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
			delete(gitopsConfigMap.ObjectMeta.Labels, types.ExcludeLabel) // remove existing velero exclude label (if exists)
			_, err = clientset.CoreV1().ConfigMaps(deployOptions.Namespace).Update(context.TODO(), gitopsConfigMap, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "failed to update kotsadm-gitops configmap in namespace %s", gitopsConfigMap.ObjectMeta.Namespace)
			}
		}
	}

	return nil
}

func readDeployOptionsFromCluster(namespace string, kubernetesConfigFlags *genericclioptions.ConfigFlags, clientset *kubernetes.Clientset) (*types.DeployOptions, error) {
	deployOptions := types.DeployOptions{
		Namespace:             namespace,
		KubernetesConfigFlags: kubernetesConfigFlags,
		ServiceType:           "ClusterIP",
	}

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
		sharedPassword, err := promptForSharedPassword()
		if err != nil {
			return nil, errors.Wrap(err, "failed to prompt for shared password")
		}

		deployOptions.SharedPassword = sharedPassword
	}

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

	// postgres password, read from the secret or create new password
	pgSecret, err := getPostgresSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres secret")
	}
	if pgSecret != nil {
		password, ok := pgSecret.Data["password"]
		if ok {
			deployOptions.PostgresPassword = string(password)
		}
	}

	// API encryption key, read from the secret or create new password
	encyptionSecret, err := getAPIEncryptionSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres secret")
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

	return &deployOptions, nil
}

func GetKotsadmOptionsFromCluster(namespace string, clientset *kubernetes.Clientset) (types.KotsadmOptions, error) {
	kotsadmOptions := types.KotsadmOptions{}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), types.KotsadmConfigMap, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return kotsadmOptions, nil
		}
		return kotsadmOptions, errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	endpoint := configMap.Data["kotsadm-registry"]
	if endpoint == "" {
		return kotsadmOptions, nil
	}

	parts := strings.Split(endpoint, "/")
	kotsadmOptions.OverrideRegistry = parts[0]
	if len(parts) == 2 {
		kotsadmOptions.OverrideNamespace = parts[1]
	} else if len(parts) > 2 {
		return kotsadmOptions, errors.Errorf("too many parts in endpoint %s", endpoint)
	}

	imagePullSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), types.PrivateKotsadmRegistrySecret, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return kotsadmOptions, nil
		}
		return kotsadmOptions, errors.Wrap(err, "failed to get existing private kotsadm registry secret")
	}

	dockerConfigJson := imagePullSecret.Data[".dockerconfigjson"]
	if len(dockerConfigJson) == 0 {
		return kotsadmOptions, nil
	}

	username, password, err := registry.GetCredentialsForRegistry(string(dockerConfigJson), kotsadmOptions.OverrideRegistry)
	if err != nil {
		return kotsadmOptions, errors.Wrap(err, "failed to parse dockerconfigjson")
	}

	kotsadmOptions.Username = username
	kotsadmOptions.Password = password
	return kotsadmOptions, nil
}
