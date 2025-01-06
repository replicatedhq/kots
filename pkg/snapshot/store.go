package snapshot

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	gcpstorage "cloud.google.com/go/storage"
	storagemgmt "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	kotss3 "github.com/replicatedhq/kots/pkg/s3"
	"github.com/replicatedhq/kots/pkg/snapshot/providers"
	"github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/replicatedhq/kots/pkg/util"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"google.golang.org/api/option"
	"gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultBackupStorageLocationName = "default"
	CloudCredentialsSecretName       = "cloud-credentials"
	RegistryImagePullSecretName      = "registry-credentials"
	SnapshotMigrationArtifactName    = "kotsadm-velero-migration"
	SnapshotStoreHostPathProvider    = "replicated.com/hostpath"
	SnapshotStoreNFSProvider         = "replicated.com/nfs"
	SnapshotStorePVCProvider         = "replicated.com/pvc"
	SnapshotStorePVCBucket           = "velero-internal-snapshots"
)

type ConfigureStoreOptions struct {
	Provider   string
	Bucket     string
	Path       string
	CACertData []byte

	AWS        *types.StoreAWS
	Google     *types.StoreGoogle
	Azure      *types.StoreAzure
	Other      *types.StoreOther
	Internal   bool
	FileSystem *types.FileSystemConfig

	KotsadmNamespace string
	RegistryConfig   *kotsadmtypes.RegistryConfig

	// If set to true, will validate the endpoint and the bucket using a pod instead (when applicable).
	// Will be ignored if SkipValidation is set to true.
	ValidateUsingAPod bool
	SkipValidation    bool
	IsMinioDisabled   bool
}

type ValidateStoreOptions struct {
	KotsadmNamespace string
	RegistryConfig   *kotsadmtypes.RegistryConfig
	CACertData       []byte
	// If set to true, will validate the endpoint and the bucket using a pod instead (when applicable)
	ValidateUsingAPod bool
}

type InvalidStoreDataError struct {
	Message string
}

func (e *InvalidStoreDataError) Error() string {
	return e.Message
}

func ConfigureStore(ctx context.Context, options ConfigureStoreOptions) (*types.Store, error) {
	existingStore, err := GetGlobalStore(ctx, options.KotsadmNamespace, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get store")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	// build a new store with the new configuration
	newStore, needsVeleroRestart, err := buildNewStore(ctx, clientset, existingStore, options)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update existing store")
	}

	// validate the new store
	if !options.SkipValidation {
		validateStoreOptions := ValidateStoreOptions{
			KotsadmNamespace:  options.KotsadmNamespace,
			RegistryConfig:    options.RegistryConfig,
			ValidateUsingAPod: options.ValidateUsingAPod,
			CACertData:        options.CACertData,
		}
		if err := validateStore(ctx, newStore, validateStoreOptions); err != nil {
			return nil, &InvalidStoreDataError{Message: errors.Cause(err).Error()}
		}
	}

	// update/create the store in the cluster
	updatedBSL, err := upsertGlobalStore(ctx, newStore, options.KotsadmNamespace, options.RegistryConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update global store")
	}

	// if a registry is configured, ensure the image pull secret is present
	if err := ensureImagePullSecret(ctx, clientset, updatedBSL.Namespace, options.RegistryConfig); err != nil {
		return nil, errors.Wrap(err, "failed to ensure image pull secret")
	}

	if err := ensureSecretsReferences(ctx, clientset, updatedBSL.Namespace); err != nil {
		return nil, errors.Wrap(err, "failed to ensure secrets references")
	}

	if err := resetRepositories(ctx, updatedBSL.Namespace); err != nil {
		return nil, errors.Wrap(err, "failed to try to reset repositories")
	}

	if needsVeleroRestart {
		// most plugins (except for local-volume-provider) require that velero be restared after updating
		if err := restartVelero(ctx, options.KotsadmNamespace); err != nil {
			return nil, errors.Wrap(err, "failed to try to restart velero")
		}
	}

	updatedStore, err := GetGlobalStore(ctx, options.KotsadmNamespace, updatedBSL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update store")
	}
	if updatedStore == nil {
		return nil, errors.New("store not found")
	}

	if err := Redact(updatedStore); err != nil {
		return nil, errors.Wrap(err, "failed to redact")
	}

	return updatedStore, nil
}

func buildNewStore(ctx context.Context, clientset kubernetes.Interface, existingStore *types.Store, options ConfigureStoreOptions) (*types.Store, bool, error) {
	store := existingStore
	if store == nil {
		store = &types.Store{}
	}

	oldBucket := store.Bucket
	needsVeleroRestart := true

	store.Provider = options.Provider
	store.Bucket = options.Bucket
	store.Path = options.Path
	store.CACertData = options.CACertData

	if options.AWS != nil {
		if store.AWS == nil {
			store.AWS = &types.StoreAWS{}
		}
		store.Azure = nil
		store.Google = nil
		store.Other = nil
		store.Internal = nil
		store.FileSystem = nil

		store.AWS.Region = options.AWS.Region
		store.AWS.UseInstanceRole = options.AWS.UseInstanceRole
		if store.AWS.UseInstanceRole {
			store.AWS.AccessKeyID = ""
			store.AWS.SecretAccessKey = ""
		} else {
			if options.AWS.AccessKeyID != "" {
				store.AWS.AccessKeyID = options.AWS.AccessKeyID
			}
			if options.AWS.SecretAccessKey != "" {
				if strings.Contains(options.AWS.SecretAccessKey, "REDACTED") {
					return nil, false, &InvalidStoreDataError{Message: "invalid aws secret access key"}
				}
				store.AWS.SecretAccessKey = options.AWS.SecretAccessKey
			}
			if store.AWS.AccessKeyID == "" || store.AWS.SecretAccessKey == "" || store.AWS.Region == "" {
				return nil, false, &InvalidStoreDataError{Message: "missing access key id and/or secret access key and/or region"}
			}
		}

	} else if options.Google != nil {
		if store.Google == nil {
			store.Google = &types.StoreGoogle{}
		}
		store.AWS = nil
		store.Azure = nil
		store.Other = nil
		store.Internal = nil
		store.FileSystem = nil

		store.Google.UseInstanceRole = options.Google.UseInstanceRole
		if store.Google.UseInstanceRole {
			store.Google.JSONFile = ""
			if options.Google.ServiceAccount != "" {
				store.Google.ServiceAccount = options.Google.ServiceAccount
			}
		} else {
			if options.Google.JSONFile != "" {
				if strings.Contains(options.Google.JSONFile, "REDACTED") {
					return nil, false, &InvalidStoreDataError{Message: "invalid JSON file"}
				}
				store.Google.JSONFile = options.Google.JSONFile
			}
		}

		if store.Google.UseInstanceRole {
			if store.Google.ServiceAccount == "" {
				return nil, false, &InvalidStoreDataError{Message: "missing service account"}
			}
		} else {
			if store.Google.JSONFile == "" {
				return nil, false, &InvalidStoreDataError{Message: "missing JSON file"}
			}
		}

	} else if options.Azure != nil {
		if store.Azure == nil {
			store.Azure = &types.StoreAzure{}
		}
		store.AWS = nil
		store.Google = nil
		store.Other = nil
		store.Internal = nil
		store.FileSystem = nil

		if options.Azure.ResourceGroup != "" {
			store.Azure.ResourceGroup = options.Azure.ResourceGroup
		}
		if options.Azure.SubscriptionID != "" {
			store.Azure.SubscriptionID = options.Azure.SubscriptionID
		}
		if options.Azure.TenantID != "" {
			store.Azure.TenantID = options.Azure.TenantID
		}
		if options.Azure.ClientID != "" {
			store.Azure.ClientID = options.Azure.ClientID
		}
		if options.Azure.ClientSecret != "" {
			if strings.Contains(options.Azure.ClientSecret, "REDACTED") {
				return nil, false, &InvalidStoreDataError{Message: "invalid client secret"}
			}
			store.Azure.ClientSecret = options.Azure.ClientSecret
		}
		if options.Azure.CloudName != "" {
			store.Azure.CloudName = options.Azure.CloudName
		}
		if options.Azure.StorageAccount != "" {
			store.Azure.StorageAccount = options.Azure.StorageAccount
		}

	} else if options.Other != nil {
		if store.Other == nil {
			store.Other = &types.StoreOther{}
		}
		store.AWS = nil
		store.Google = nil
		store.Azure = nil
		store.Internal = nil
		store.FileSystem = nil

		store.Provider = "aws"
		if options.Other.AccessKeyID != "" {
			store.Other.AccessKeyID = options.Other.AccessKeyID
		}
		if options.Other.SecretAccessKey != "" {
			if strings.Contains(options.Other.SecretAccessKey, "REDACTED") {
				return nil, false, &InvalidStoreDataError{Message: "invalid secret access key"}
			}
			store.Other.SecretAccessKey = options.Other.SecretAccessKey
		}
		if options.Other.Region != "" {
			store.Other.Region = options.Other.Region
		}
		if options.Other.Endpoint != "" {
			store.Other.Endpoint = options.Other.Endpoint
		}
		if store.Other.AccessKeyID == "" || store.Other.SecretAccessKey == "" || store.Other.Endpoint == "" || store.Other.Region == "" {
			return nil, false, &InvalidStoreDataError{Message: "access key, secret key, endpoint and region are required"}
		}
	} else if options.Internal && !options.IsMinioDisabled {
		isKurl, err := kurl.IsKurl(clientset)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to check if cluster is kurl")
		}

		if !isKurl {
			return nil, false, &InvalidStoreDataError{Message: "cannot use internal storage on a non-kurl cluster"}
		}

		if store.Internal == nil {
			store.Internal = &types.StoreInternal{}
		}
		store.AWS = nil
		store.Google = nil
		store.Azure = nil
		store.Other = nil
		store.FileSystem = nil

		secret, err := kotsutil.GetKurlS3Secret()
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to get s3 secret")
		}
		if secret == nil {
			return nil, false, errors.New("s3 secret does not exist")
		}

		store.Provider = "aws"
		store.Bucket = string(secret.Data["velero-local-bucket"])
		store.Path = ""

		store.Internal.AccessKeyID = string(secret.Data["access-key-id"])
		store.Internal.SecretAccessKey = string(secret.Data["secret-access-key"])
		store.Internal.Endpoint = string(secret.Data["endpoint"])
		store.Internal.ObjectStoreClusterIP = string(secret.Data["object-store-cluster-ip"])
		store.Internal.Region = "us-east-1"
	} else if options.Internal && options.IsMinioDisabled {
		isKurl, err := kurl.IsKurl(clientset)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to check if cluster is kurl")
		}

		if !isKurl {
			return nil, false, &InvalidStoreDataError{Message: "cannot use internal storage on a non-kurl cluster"}
		}

		if store.Internal == nil {
			store.Internal = &types.StoreInternal{}
		}
		store.AWS = nil
		store.Google = nil
		store.Azure = nil
		store.Other = nil
		store.FileSystem = nil

		store.Provider = SnapshotStorePVCProvider
		store.Bucket = SnapshotStorePVCBucket
	} else if options.FileSystem != nil && !options.IsMinioDisabled {
		// Legacy Minio Provider

		store.AWS = nil
		store.Google = nil
		store.Azure = nil
		store.Other = nil
		store.Internal = nil

		store.Provider = FileSystemMinioProvider
		store.Bucket = FileSystemMinioBucketName
		store.Path = ""

		storeFileSystem, err := BuildMinioStoreFileSystem(ctx, clientset, options.KotsadmNamespace)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to build file system store")
		}
		store.FileSystem = storeFileSystem
	} else if options.FileSystem != nil && options.IsMinioDisabled {
		store.AWS = nil
		store.Google = nil
		store.Azure = nil
		store.Other = nil
		store.Internal = nil

		newBucket, err := GetLvpBucket(options.FileSystem)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to generate bucket name")
		}

		if oldBucket != newBucket {
			// bucket has changed, so the plugin will handle the restart
			needsVeleroRestart = false
		}

		store.Bucket = newBucket
		store.Provider = GetLvpProvider(options.FileSystem)

		if isMinioMigration(clientset, options.KotsadmNamespace) {
			store.Path = "/velero"
		}

		store.FileSystem = BuildLvpStoreFileSystem(options.FileSystem)
	}

	return store, needsVeleroRestart, nil
}

// upsertGlobalStore will update the in-cluster storage with exactly what's in the store param
func upsertGlobalStore(ctx context.Context, store *types.Store, kotsadmNamespace string, registryConfig *kotsadmtypes.RegistryConfig) (*velerov1.BackupStorageLocation, error) {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero client")
	}

	bsl, err := FindBackupStoreLocation(ctx, clientset, veleroClient, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	if bsl == nil {
		veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, kotsadmNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to detect velero namespace")
		}
		if veleroNamespace == "" {
			return nil, errors.New("velero is not installed")
		}
		bsl = &velerov1.BackupStorageLocation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      DefaultBackupStorageLocationName,
				Namespace: veleroNamespace,
			},
			Spec: velerov1.BackupStorageLocationSpec{
				Default: true,
			},
		}
	}

	bsl.Spec.Provider = store.Provider
	bsl.Spec.Config = map[string]string{} // Ensure config is clean when switching providers

	if bsl.Spec.ObjectStorage == nil {
		bsl.Spec.ObjectStorage = &velerov1.ObjectStorageLocation{}
	}
	bsl.Spec.ObjectStorage.Bucket = store.Bucket
	bsl.Spec.ObjectStorage.Prefix = store.Path
	bsl.Spec.ObjectStorage.CACert = store.CACertData

	if store.AWS != nil {
		resolver := endpoints.DefaultResolver()
		resolvedEndpoint, err := resolver.EndpointFor("s3", store.AWS.Region)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve endpoint")
		}
		bsl.Spec.Config = map[string]string{
			"region": store.AWS.Region,
			"s3Url":  resolvedEndpoint.URL,
		}

		awsCredentials, err := BuildAWSCredentials(store.AWS.AccessKeyID, store.AWS.SecretAccessKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to format aws credentials")
		}

		if err := ensureCloudCredentialsSecret(ctx, clientset, bsl.Namespace, awsCredentials); err != nil {
			return nil, errors.Wrap(err, "failed to ensure cloud credentials secret")
		}
	} else if store.Other != nil {
		bsl.Spec.Config = map[string]string{
			"region":           store.Other.Region,
			"s3Url":            store.Other.Endpoint,
			"s3ForcePathStyle": "true",
		}

		otherCredentials, err := BuildAWSCredentials(store.Other.AccessKeyID, store.Other.SecretAccessKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to format other credentials")
		}

		if err := ensureCloudCredentialsSecret(ctx, clientset, bsl.Namespace, otherCredentials); err != nil {
			return nil, errors.Wrap(err, "failed to ensure cloud credentials secret")
		}
	} else if store.Internal != nil {
		isMinioDisabled, err := IsFileSystemMinioDisabled(kotsadmNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check for existing snapshot preference")
		}

		if !isMinioDisabled {
			bsl.Spec.Config = map[string]string{
				"region":           store.Internal.Region,
				"s3Url":            store.Internal.Endpoint,
				"publicUrl":        fmt.Sprintf("http://%s", store.Internal.ObjectStoreClusterIP),
				"s3ForcePathStyle": "true",
			}

			internalCredentials, err := BuildAWSCredentials(store.Internal.AccessKeyID, store.Internal.SecretAccessKey)
			if err != nil {
				return nil, errors.Wrap(err, "failed to format internal credentials")
			}

			if err := ensureCloudCredentialsSecret(ctx, clientset, bsl.Namespace, internalCredentials); err != nil {
				return nil, errors.Wrap(err, "failed to ensure cloud credentials secret")
			}
		} else {
			bsl.Spec.Config = map[string]string{
				"storageSize":      "50Gi",
				"resticRepoPrefix": "/var/velero-local-volume-provider/velero-internal-snapshots/restic",
			}
		}
	} else if store.FileSystem != nil && store.Provider == FileSystemMinioProvider {
		// Legacy Minio case
		err = updateMinioFileSystemStore(ctx, clientset, store, bsl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update file system store for minio")
		}
	} else if store.FileSystem != nil {
		err = updateLvpFileSystemStore(ctx, store, bsl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update file system store for lvp")
		}
	} else if store.Google != nil {
		if store.Google.UseInstanceRole {
			bsl.Spec.Config = map[string]string{
				"serviceAccount": store.Google.ServiceAccount,
			}

			// delete the secret
			err := clientset.CoreV1().Secrets(bsl.Namespace).Delete(ctx, CloudCredentialsSecretName, metav1.DeleteOptions{})
			if err != nil && !kuberneteserrors.IsNotFound(err) {
				return nil, errors.Wrap(err, "failed to delete google creds secret")
			}
		} else {
			if err := ensureCloudCredentialsSecret(ctx, clientset, bsl.Namespace, []byte(store.Google.JSONFile)); err != nil {
				return nil, errors.Wrap(err, "failed to ensure cloud credentials secret")
			}
		}
	} else if store.Azure != nil {
		// https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure/blob/main/backupstoragelocation.md
		bsl.Spec.Config = map[string]string{
			"resourceGroup":  store.Azure.ResourceGroup,
			"storageAccount": store.Azure.StorageAccount,
			"subscriptionId": store.Azure.SubscriptionID,
		}

		config := providers.Azure{
			SubscriptionID: store.Azure.SubscriptionID,
			TenantID:       store.Azure.TenantID,
			ClientID:       store.Azure.ClientID,
			ClientSecret:   store.Azure.ClientSecret,
			ResourceGroup:  store.Azure.ResourceGroup,
			CloudName:      store.Azure.CloudName,
		}
		if err := ensureCloudCredentialsSecret(ctx, clientset, bsl.Namespace, providers.RenderAzureConfig(config)); err != nil {
			return nil, errors.Wrap(err, "failed to ensure cloud credentials secret")
		}
	} else {
		return nil, errors.Wrap(err, "malformed input - could not determine provider")
	}

	updated, err := upsertBackupStorageLocation(ctx, bsl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to upsert backup storage location")
	}

	return updated, nil
}

func ensureCloudCredentialsSecret(ctx context.Context, clientset kubernetes.Interface, veleroNamespace string, creds []byte) error {
	credsSecret, err := clientset.CoreV1().Secrets(veleroNamespace).Get(ctx, CloudCredentialsSecretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read secret")
	}

	if kuberneteserrors.IsNotFound(err) {
		toCreate := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      CloudCredentialsSecretName,
				Namespace: veleroNamespace,
			},
			Data: map[string][]byte{
				"cloud": creds,
			},
		}
		_, err := clientset.CoreV1().Secrets(veleroNamespace).Create(ctx, toCreate, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}
	} else {
		if credsSecret.Data == nil {
			credsSecret.Data = map[string][]byte{}
		}
		credsSecret.Data["cloud"] = creds

		if _, err := clientset.CoreV1().Secrets(veleroNamespace).Update(ctx, credsSecret, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "failed to update secret")
		}
	}

	return nil
}

func ensureImagePullSecret(ctx context.Context, clientset kubernetes.Interface, veleroNamespace string, registryConfig *kotsadmtypes.RegistryConfig) error {
	if registryConfig == nil ||
		registryConfig.OverrideRegistry == "" ||
		registryConfig.Username == "" ||
		registryConfig.Password == "" {
		return nil
	}

	type DockercfgAuth struct {
		Auth string `json:"auth,omitempty"`
	}
	type DockerCfgJSON struct {
		Auths map[string]DockercfgAuth `json:"auths"`
	}

	host := strings.Split(registryConfig.OverrideRegistry, "/")[0]

	dockercfgAuth := DockercfgAuth{
		Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", registryConfig.Username, registryConfig.Password))),
	}
	dockerCfgJSON := DockerCfgJSON{
		Auths: map[string]DockercfgAuth{
			host: dockercfgAuth,
		},
	}

	secretData, err := json.Marshal(dockerCfgJSON)
	if err != nil {
		return errors.Wrap(err, "failed to marshal pull secret data")
	}

	imagePullSecret, err := clientset.CoreV1().Secrets(veleroNamespace).Get(ctx, RegistryImagePullSecretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read secret")
	}

	if kuberneteserrors.IsNotFound(err) {
		toCreate := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      RegistryImagePullSecretName,
				Namespace: veleroNamespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": secretData,
			},
		}
		_, err := clientset.CoreV1().Secrets(veleroNamespace).Create(ctx, toCreate, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}
	} else {
		if imagePullSecret.Data == nil {
			imagePullSecret.Data = map[string][]byte{}
		}
		imagePullSecret.Data[".dockerconfigjson"] = secretData

		if _, err := clientset.CoreV1().Secrets(veleroNamespace).Update(ctx, imagePullSecret, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "failed to update secret")
		}
	}

	return nil
}

func ensureSecretsReferences(ctx context.Context, clientset kubernetes.Interface, veleroNamespace string) error {
	veleroDeployment, err := clientset.AppsV1().Deployments(veleroNamespace).Get(ctx, "velero", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get velero deployment")
	}

	nodeAgentDaemonset, err := clientset.AppsV1().DaemonSets(veleroNamespace).Get(ctx, "node-agent", metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get node-agent daemonset")
	}
	if kuberneteserrors.IsNotFound(err) {
		// check the old name ("restic") for backwards compatibility
		nodeAgentDaemonset, err = clientset.AppsV1().DaemonSets(veleroNamespace).Get(ctx, "restic", metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get restic daemonset")
		}
	}

	_, err = clientset.CoreV1().Secrets(veleroNamespace).Get(ctx, CloudCredentialsSecretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read cloud credentials secret")
	}
	if err == nil {
		// ensure that velero and node-agent have the cloud credentials secret mounted
		veleroDeployment.Spec.Template.Spec.Volumes = k8sutil.MergeVolumes(cloudCredentialsVolumes(), veleroDeployment.Spec.Template.Spec.Volumes, false)
		veleroDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = k8sutil.MergeVolumeMounts(cloudCredentialsVolumeMounts(), veleroDeployment.Spec.Template.Spec.Containers[0].VolumeMounts, false)
		veleroDeployment.Spec.Template.Spec.Containers[0].Env = k8sutil.MergeEnvVars(cloudCredentialsEnvVars(), veleroDeployment.Spec.Template.Spec.Containers[0].Env, false)

		nodeAgentDaemonset.Spec.Template.Spec.Volumes = k8sutil.MergeVolumes(cloudCredentialsVolumes(), nodeAgentDaemonset.Spec.Template.Spec.Volumes, false)
		nodeAgentDaemonset.Spec.Template.Spec.Containers[0].VolumeMounts = k8sutil.MergeVolumeMounts(cloudCredentialsVolumeMounts(), nodeAgentDaemonset.Spec.Template.Spec.Containers[0].VolumeMounts, false)
		nodeAgentDaemonset.Spec.Template.Spec.Containers[0].Env = k8sutil.MergeEnvVars(cloudCredentialsEnvVars(), nodeAgentDaemonset.Spec.Template.Spec.Containers[0].Env, false)
	}

	_, err = clientset.CoreV1().Secrets(veleroNamespace).Get(ctx, RegistryImagePullSecretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read image pull secret")
	}
	if err == nil {
		// ensure that velero and node-agent have the image pull secret referenced
		veleroDeployment.Spec.Template.Spec.ImagePullSecrets = k8sutil.MergeImagePullSecrets([]corev1.LocalObjectReference{{Name: RegistryImagePullSecretName}}, veleroDeployment.Spec.Template.Spec.ImagePullSecrets, false)
		nodeAgentDaemonset.Spec.Template.Spec.ImagePullSecrets = k8sutil.MergeImagePullSecrets([]corev1.LocalObjectReference{{Name: RegistryImagePullSecretName}}, nodeAgentDaemonset.Spec.Template.Spec.ImagePullSecrets, false)
	}

	if _, err := clientset.AppsV1().Deployments(veleroNamespace).Update(ctx, veleroDeployment, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update velero deployment")
	}

	if _, err := clientset.AppsV1().DaemonSets(veleroNamespace).Update(ctx, nodeAgentDaemonset, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed to update node-agent daemonset")
	}

	return nil
}

func cloudCredentialsVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "cloud-credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: CloudCredentialsSecretName,
				},
			},
		},
	}
}

func cloudCredentialsVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "cloud-credentials",
			MountPath: "/credentials",
		},
	}
}

func cloudCredentialsEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: "/credentials/cloud",
		},
		{
			Name:  "AWS_SHARED_CREDENTIALS_FILE",
			Value: "/credentials/cloud",
		},
		{
			Name:  "AZURE_CREDENTIALS_FILE",
			Value: "/credentials/cloud",
		},
		{
			Name:  "ALIBABA_CLOUD_CREDENTIALS_FILE",
			Value: "/credentials/cloud",
		},
	}
}

func upsertBackupStorageLocation(ctx context.Context, bsl *velerov1.BackupStorageLocation) (*velerov1.BackupStorageLocation, error) {
	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero client")
	}

	err = veleroClient.Update(ctx, bsl)
	if err == nil {
		return bsl, nil
	}

	if kuberneteserrors.IsNotFound(err) {
		err = veleroClient.Create(ctx, bsl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create backup storage location")
		}
		return bsl, nil
	}

	return nil, errors.Wrap(err, "failed to get backup storage location")
}

func updateMinioFileSystemStore(ctx context.Context, clientset kubernetes.Interface, store *types.Store, bsl *velerov1.BackupStorageLocation) error {
	bsl.Spec.Config = map[string]string{
		"region":           store.FileSystem.Region,
		"s3Url":            store.FileSystem.Endpoint,
		"publicUrl":        fmt.Sprintf("http://%s:%d", store.FileSystem.ObjectStoreClusterIP, FileSystemMinioServicePort),
		"s3ForcePathStyle": "true",
	}

	fileSystemCredentials, err := BuildAWSCredentials(store.FileSystem.AccessKeyID, store.FileSystem.SecretAccessKey)
	if err != nil {
		return errors.Wrap(err, "failed to format file system credentials")
	}

	if err := ensureCloudCredentialsSecret(ctx, clientset, bsl.Namespace, fileSystemCredentials); err != nil {
		return errors.Wrap(err, "failed to ensure cloud credentials secret")
	}

	return nil
}

func updateLvpFileSystemStore(ctx context.Context, store *types.Store, bsl *velerov1.BackupStorageLocation) error {
	if store.FileSystem.Config == nil {
		return errors.New("missing file system config")
	}

	resticDir := path.Join(store.Bucket, store.Path)
	if store.FileSystem.Config.HostPath != nil && *store.FileSystem.Config.HostPath != "" {
		bsl.Spec.Config = map[string]string{
			"path":             *store.FileSystem.Config.HostPath,
			"resticRepoPrefix": fmt.Sprintf("/var/velero-local-volume-provider/%s/restic", resticDir),
		}
	} else {
		if p := store.FileSystem.Config.NFS.Path; p == "" {
			store.FileSystem.Config.NFS.Path = "/"
		}

		bsl.Spec.Config = map[string]string{
			"path":             store.FileSystem.Config.NFS.Path,
			"server":           store.FileSystem.Config.NFS.Server,
			"resticRepoPrefix": fmt.Sprintf("/var/velero-local-volume-provider/%s/restic", resticDir),
		}
	}

	// This will force an immediate sync by the velero controller
	bsl.Status.LastSyncedTime = nil
	return nil
}

// GetGlobalStore will return the global store from the current backup storage location
// or will find it, if the param is nil
func GetGlobalStore(ctx context.Context, kotsadmNamespace string, bsl *velerov1.BackupStorageLocation) (*types.Store, error) {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero client")
	}

	if bsl == nil {
		bsl, err = FindBackupStoreLocation(ctx, clientset, veleroClient, kotsadmNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find backupstoragelocations")
		}
		if bsl == nil {
			return nil, nil
		}
	}

	if bsl.Spec.ObjectStorage == nil {
		return nil, nil
	}

	store := types.Store{
		Provider:   bsl.Spec.Provider,
		Bucket:     bsl.Spec.ObjectStorage.Bucket,
		Path:       bsl.Spec.ObjectStorage.Prefix,
		CACertData: bsl.Spec.ObjectStorage.CACert,
	}

	switch store.Provider {
	case "aws":
		err := mapAWSBackupStorageLocationToStore(bsl, &store)
		if err != nil {
			return nil, errors.Wrap(err, "failed to map aws backup storage location to store")
		}

		awsSecret, err := clientset.CoreV1().Secrets(bsl.Namespace).Get(ctx, "cloud-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to read aws secret")
		}

		if err == nil {
			awsCfg, err := ini.Load(awsSecret.Data["cloud"])
			if err != nil {
				return nil, errors.Wrap(err, "failed to load aws credentials")
			}

			for _, section := range awsCfg.Sections() {
				if section.Name() == "default" {
					if store.Internal != nil {
						store.Internal.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.Internal.SecretAccessKey = section.Key("aws_secret_access_key").Value()
					} else if store.FileSystem != nil {
						store.FileSystem.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.FileSystem.SecretAccessKey = section.Key("aws_secret_access_key").Value()
					} else if store.Other != nil {
						store.Other.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.Other.SecretAccessKey = section.Key("aws_secret_access_key").Value()
					} else if store.AWS != nil {
						store.AWS.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.AWS.SecretAccessKey = section.Key("aws_secret_access_key").Value()
						if store.AWS.AccessKeyID == "" && store.AWS.SecretAccessKey == "" {
							store.AWS.UseInstanceRole = true // cloud-credentials present, values empty, assume instance role
						}
					}
				}
			}
		}

	case "azure":
		// TODO validate these keys in a real azure account
		store.Azure = &types.StoreAzure{
			ResourceGroup:  bsl.Spec.Config["resourceGroup"],
			StorageAccount: bsl.Spec.Config["storageAccount"],
			SubscriptionID: bsl.Spec.Config["subscriptionId"],
		}

		// get the secret
		azureSecret, err := clientset.CoreV1().Secrets(bsl.Namespace).Get(ctx, "cloud-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to read azure secret")
		}

		if err == nil {
			azureConfig := providers.ParseAzureConfig(azureSecret.Data["cloud"])
			store.Azure.TenantID = azureConfig.TenantID
			store.Azure.ClientID = azureConfig.ClientID
			store.Azure.ClientSecret = azureConfig.ClientSecret
			store.Azure.CloudName = azureConfig.CloudName
		}

		if store.Azure.CloudName == "" {
			store.Azure.CloudName = providers.AzureDefaultCloud
		}

	case "gcp":
		currentSecret, err := clientset.CoreV1().Secrets(bsl.Namespace).Get(ctx, "cloud-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to read google secret")
		}

		jsonFile := ""
		if err == nil {
			currentJSONFile, ok := currentSecret.Data["cloud"]
			if ok {
				jsonFile = string(currentJSONFile)
			}
		}

		store.Google = &types.StoreGoogle{
			ServiceAccount:  bsl.Spec.Config["serviceAccount"],
			JSONFile:        jsonFile,
			UseInstanceRole: jsonFile == "",
		}
	case SnapshotStoreHostPathProvider:
		path := bsl.Spec.Config["path"]
		store.FileSystem = &types.StoreFileSystem{
			Config: &types.FileSystemConfig{
				HostPath: &path,
			},
		}
	case SnapshotStoreNFSProvider:
		store.FileSystem = &types.StoreFileSystem{
			Config: &types.FileSystemConfig{
				NFS: &types.NFSConfig{
					Path:   bsl.Spec.Config["path"],
					Server: bsl.Spec.Config["server"],
				},
			},
		}
	case SnapshotStorePVCProvider:
		store.Internal = &types.StoreInternal{}
	}

	return &store, nil
}

func mapAWSBackupStorageLocationToStore(kotsadmVeleroBackendStorageLocation *velerov1.BackupStorageLocation, store *types.Store) error {
	endpoint, isS3Compatible := kotsadmVeleroBackendStorageLocation.Spec.Config["s3Url"]
	u, err := url.Parse(endpoint)
	if err != nil {
		return errors.Wrap(err, "failed to parse s3 url")
	}
	// without endpoint, the ui has no logic to figure if it is amazon-s3 or other-s3 compatible storages
	shouldMapToAWS := !isS3Compatible || strings.HasSuffix(u.Hostname(), ".amazonaws.com")
	if util.IsEmbeddedCluster() {
		// embedded clusters only support other s3 compatible storage
		shouldMapToAWS = false
	}
	if shouldMapToAWS {
		store.AWS = &types.StoreAWS{
			Region: kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
		}
		return nil
	}

	// check if using kurl internal store
	s3Secret, err := kotsutil.GetKurlS3Secret()
	if err == nil && s3Secret != nil && string(s3Secret.Data["endpoint"]) == endpoint {
		store.Internal = &types.StoreInternal{
			Region:               kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
			Endpoint:             endpoint,
			ObjectStoreClusterIP: string(s3Secret.Data["object-store-cluster-ip"]),
		}
		return nil
	}

	// check if using file system store
	serviceName := strings.Split(u.Hostname(), ".")[0]
	if u.Scheme == "http" && serviceName == FileSystemMinioServiceName {
		publicURL, ok := kotsadmVeleroBackendStorageLocation.Spec.Config["publicUrl"]
		if !ok {
			return errors.New("public url for file system store not found")
		}
		u, err := url.Parse(publicURL)
		if err != nil {
			return errors.Wrap(err, "failed to parse public url for file system store")
		}
		store.FileSystem = &types.StoreFileSystem{
			Region:               kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
			Endpoint:             endpoint,
			ObjectStoreClusterIP: u.Hostname(),
		}
		return nil
	}

	store.Other = &types.StoreOther{
		Region:   kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
		Endpoint: endpoint,
	}

	return nil
}

// FindBackupStoreLocation will find the backup storage location used by velero
// kotsadmNamespace is only required in minimal rbac installations. if empty, cluster scope privileges will be needed to detect and validate velero
func FindBackupStoreLocation(ctx context.Context, clientset kubernetes.Interface, ctrlClient kbclient.Client, kotsadmNamespace string) (*velerov1.BackupStorageLocation, error) {
	veleroNamespace, err := DetectVeleroNamespace(ctx, clientset, kotsadmNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect velero namespace")
	}

	if veleroNamespace == "" {
		return nil, nil
	}

	var backupStorageLocations velerov1.BackupStorageLocationList
	err = ctrlClient.List(ctx, &backupStorageLocations, kbclient.InNamespace(veleroNamespace))
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to list backupstoragelocations")
	}

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == DefaultBackupStorageLocationName {
			return &backupStorageLocation, nil
		}
	}

	return nil, nil
}

// UpdateBackupStorageLocation applies an updated Velero backup storage location resource to the cluster
func UpdateBackupStorageLocation(ctx context.Context, veleroNamespace string, bsl *velerov1.BackupStorageLocation) error {
	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create velero client")
	}

	err = veleroClient.Update(ctx, bsl)
	if err != nil {
		return errors.Wrap(err, "failed to update backupstoragelocation")
	}

	return nil
}

func BuildAWSCredentials(accessKeyID, secretAccessKey string) ([]byte, error) {
	awsCfg := ini.Empty()
	section, err := awsCfg.NewSection("default")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create default section in aws creds")
	}
	_, err = section.NewKey("aws_access_key_id", accessKeyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create access key")
	}

	_, err = section.NewKey("aws_secret_access_key", secretAccessKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create secret access key")
	}

	var awsCredentials bytes.Buffer
	writer := bufio.NewWriter(&awsCredentials)
	_, err = awsCfg.WriteTo(writer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write ini")
	}
	if err := writer.Flush(); err != nil {
		return nil, errors.Wrap(err, "failed to flush buffer")
	}

	return awsCredentials.Bytes(), nil
}

func BuildMinioStoreFileSystem(ctx context.Context, clientset kubernetes.Interface, kotsadmNamespace string) (*types.StoreFileSystem, error) {
	secret, err := clientset.CoreV1().Secrets(kotsadmNamespace).Get(ctx, FileSystemMinioSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file system minio secret")
	}

	service, err := clientset.CoreV1().Services(kotsadmNamespace).Get(ctx, FileSystemMinioServiceName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get file system minio service")
	}

	storeFileSystem := types.StoreFileSystem{}
	storeFileSystem.AccessKeyID = string(secret.Data["MINIO_ACCESS_KEY"])
	storeFileSystem.SecretAccessKey = string(secret.Data["MINIO_SECRET_KEY"])
	storeFileSystem.Endpoint = fmt.Sprintf("http://%s.%s:%d", FileSystemMinioServiceName, kotsadmNamespace, service.Spec.Ports[0].Port)
	storeFileSystem.ObjectStoreClusterIP = service.Spec.ClusterIP
	storeFileSystem.Region = FileSystemMinioRegion

	return &storeFileSystem, nil
}

func BuildLvpStoreFileSystem(config *types.FileSystemConfig) *types.StoreFileSystem {
	storeFileSystem := types.StoreFileSystem{}

	if config.NFS != nil && config.NFS.Path == "" {
		config.NFS.Path = "/"
	}
	storeFileSystem.Config = config

	return &storeFileSystem
}

func validateStore(ctx context.Context, store *types.Store, options ValidateStoreOptions) error {
	if store.AWS != nil {
		if err := validateAWS(store.AWS, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate AWS configuration")
		}
		return nil
	}

	if store.Azure != nil {
		if err := validateAzure(ctx, store.Azure, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate Azure configuration")
		}
		return nil
	}

	if store.Google != nil {
		if err := validateGCP(store.Google, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate GCP configuration")
		}
		return nil
	}

	if store.Other != nil {
		if err := validateOther(ctx, store.Other, store.Bucket, options); err != nil {
			return errors.Wrap(err, "failed to validate S3-compatible configuration")
		}
		return nil
	}

	// Internal with Minio
	if store.Internal != nil && store.Provider == "aws" {
		if err := validateInternalS3(ctx, store.Internal, store.Bucket, options); err != nil {
			return errors.Wrap(err, "failed to validate Internal S3 configuration")
		}
		return nil
	}

	// Internal with PVC
	if store.Internal != nil {
		if err := validateInternalPVC(store.Provider, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate Internal PVC configuration")
		}
		return nil
	}

	// Legacy Minio FS
	if store.FileSystem != nil && store.Provider == FileSystemMinioProvider {
		if err := validateMinioFileSystem(ctx, store.FileSystem, store.Bucket, options); err != nil {
			return errors.Wrap(err, "failed to validate Minio File System configuration")
		}
		return nil
	}

	if store.FileSystem != nil {
		if err := validateLvpFileSystem(ctx, store, options); err != nil {
			return errors.Wrap(err, "failed to validate LVP File System configuration")
		}
		return nil
	}

	return errors.New("no valid configuration found")
}

func validateAWS(storeAWS *types.StoreAWS, bucket string) error {
	s3Config := &aws.Config{
		Region:           aws.String(storeAWS.Region),
		DisableSSL:       aws.Bool(false),
		S3ForcePathStyle: aws.Bool(false), // TODO: this may need to be configurable
	}

	ec2Session, err := session.NewSession()
	if err != nil {
		return errors.Wrap(err, "failed to create AWS ec2 session")
	}
	if storeAWS.UseInstanceRole {
		s3Config.Credentials = credentials.NewChainCredentials([]credentials.Provider{
			&ec2rolecreds.EC2RoleProvider{
				Client:       ec2metadata.New(ec2Session),
				ExpiryWindow: 5 * time.Minute,
			},
		})
	} else {
		s3Config.Credentials = credentials.NewStaticCredentials(storeAWS.AccessKeyID, storeAWS.SecretAccessKey, "")
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return errors.Wrap(err, "failed to create AWS S3 session")
	}
	s3Client := s3.New(newSession)

	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		if err == credentials.ErrNoValidProvidersFoundInChain && storeAWS.UseInstanceRole {
			// error returned when instance does not have proper role and empty creds are passed
			return errors.New("failed to validate instance role")
		}
		return errors.Wrap(err, "bucket does not exist")
	}

	return nil
}

func validateAzure(ctx context.Context, storeAzure *types.StoreAzure, bucket string) error {
	// Mostly copied from Velero Azure plugin

	env, err := azure.EnvironmentFromName(storeAzure.CloudName)
	if err != nil {
		return errors.Wrap(err, "failed to find azure env")
	}

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, storeAzure.TenantID)
	if err != nil {
		return errors.Wrap(err, "failed to get OAuthConfig")
	}

	spt, err := adal.NewServicePrincipalToken(*oauthConfig, storeAzure.ClientID, storeAzure.ClientSecret, env.ResourceManagerEndpoint)
	if err != nil {
		return errors.Wrap(err, "failed to get service principal token")
	}

	storageAccountsClient := storagemgmt.NewAccountsClientWithBaseURI(env.ResourceManagerEndpoint, storeAzure.SubscriptionID)
	storageAccountsClient.Authorizer = autorest.NewBearerAuthorizer(spt)

	res, err := storageAccountsClient.ListKeys(ctx, storeAzure.ResourceGroup, storeAzure.StorageAccount)
	if err != nil {
		return errors.Wrap(err, "failed to list account keys")
	}
	if res.Keys == nil || len(*res.Keys) == 0 {
		return errors.New("No storage keys found")
	}

	var storageKey string
	for _, key := range *res.Keys {
		// case-insensitive comparison because the ListKeys call returns e.g. "FULL" but
		// the storagemgmt.Full constant in the SDK is defined as "Full".
		if strings.EqualFold(string(key.Permissions), string(storagemgmt.Full)) {
			storageKey = *key.Value
			break
		}
	}

	if storageKey == "" {
		return errors.New("No storage key with Full permissions found")
	}

	storageClient, err := storage.NewBasicClientOnSovereignCloud(storeAzure.StorageAccount, storageKey, env)
	if err != nil {
		return errors.Wrap(err, "failed to get storage client")
	}

	blobClient := storageClient.GetBlobService()
	container := blobClient.GetContainerReference(bucket)
	if container == nil {
		return errors.Errorf("unable to get container reference for bucket %s", bucket)
	}

	exists, err := container.Exists()
	if err != nil {
		return errors.Wrap(err, "failed to check container existence")
	}

	if !exists {
		return errors.New("container does not exist")
	}

	return nil
}

func validateGCP(storeGoogle *types.StoreGoogle, bucket string) error {
	ctx := context.Background()
	if storeGoogle.UseInstanceRole {
		// TODO: validate IAM access
	} else {
		client, err := gcpstorage.NewClient(ctx, option.WithCredentialsJSON([]byte(storeGoogle.JSONFile)))
		if err != nil {
			return errors.Wrap(err, "failed to create storage client")
		}

		objectsItr := client.Bucket(bucket).Objects(ctx, &gcpstorage.Query{})
		_, err = objectsItr.Next()
		if err != nil {
			if strings.Contains(err.Error(), "no more items in iterator") {
				return nil
			}

			return errors.Wrap(err, "failed to get bucket attributes")
		}
	}

	return nil
}

func validateOther(ctx context.Context, storeOther *types.StoreOther, bucket string, options ValidateStoreOptions) error {
	if options.ValidateUsingAPod {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return errors.Wrap(err, "failed to get k8s clientset")
		}

		podName := fmt.Sprintf("kotsadm-validate-other-s3-bucket-%d", time.Now().Unix())

		podOptions := kotss3.S3OpsPodOptions{
			PodName:         podName,
			Endpoint:        storeOther.Endpoint,
			BucketName:      bucket,
			CACertData:      options.CACertData,
			AccessKeyID:     storeOther.AccessKeyID,
			SecretAccessKey: storeOther.SecretAccessKey,
			Namespace:       options.KotsadmNamespace,
			IsOpenShift:     k8sutil.IsOpenShift(clientset),
			RegistryConfig:  options.RegistryConfig,
		}

		return kotss3.HeadS3BucketUsingAPod(ctx, clientset, podOptions)
	}

	s3Config := &aws.Config{
		Region:           aws.String(storeOther.Region),
		Endpoint:         aws.String(storeOther.Endpoint),
		DisableSSL:       aws.Bool(true), // TODO: this needs to be configurable
		S3ForcePathStyle: aws.Bool(true), // TODO: this may need to be configurable
	}

	if storeOther.AccessKeyID != "" && storeOther.SecretAccessKey != "" {
		s3Config.Credentials = credentials.NewStaticCredentials(storeOther.AccessKeyID, storeOther.SecretAccessKey, "")
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return errors.Wrap(err, "failed to create s3 session")
	}
	s3Client := s3.New(newSession)

	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		return errors.Wrap(err, "bucket does not exist")
	}

	return nil
}

func validateInternalS3(ctx context.Context, storeInternal *types.StoreInternal, bucket string, options ValidateStoreOptions) error {
	if options.ValidateUsingAPod {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return errors.Wrap(err, "failed to get k8s clientset")
		}

		podName := fmt.Sprintf("kotsadm-validate-internal-bucket-%d", time.Now().Unix())

		podOptions := kotss3.S3OpsPodOptions{
			PodName:         podName,
			Endpoint:        storeInternal.Endpoint,
			BucketName:      bucket,
			AccessKeyID:     storeInternal.AccessKeyID,
			SecretAccessKey: storeInternal.SecretAccessKey,
			Namespace:       options.KotsadmNamespace,
			IsOpenShift:     k8sutil.IsOpenShift(clientset),
			RegistryConfig:  options.RegistryConfig,
		}

		return kotss3.HeadS3BucketUsingAPod(ctx, clientset, podOptions)
	}

	s3Config := &aws.Config{
		Region:           aws.String(storeInternal.Region),
		Endpoint:         aws.String(storeInternal.Endpoint),
		DisableSSL:       aws.Bool(true), // TODO: this needs to be configurable
		S3ForcePathStyle: aws.Bool(true),
	}

	if storeInternal.AccessKeyID != "" && storeInternal.SecretAccessKey != "" {
		s3Config.Credentials = credentials.NewStaticCredentials(storeInternal.AccessKeyID, storeInternal.SecretAccessKey, "")
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return errors.Wrap(err, "failed to create s3 session")
	}
	s3Client := s3.New(newSession)

	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		return errors.Wrap(err, "bucket does not exist")
	}

	return nil
}

// validateInternalPVC checks that the internal store is configured to use PVC
func validateInternalPVC(provider string, bucket string) error {
	if provider != SnapshotStorePVCProvider {
		return fmt.Errorf("failed to validate provider %s", provider)
	}

	if bucket != SnapshotStorePVCBucket {
		return fmt.Errorf("failed to validate bucket %s", bucket)
	}

	return nil
}

// validateLvpFileSystem checks that the fs configuration can be mounted, that the chosen directory is writable and
// also test for legacy minio files. If minio files are detected, the store is updated with the /velero prefix
func validateLvpFileSystem(ctx context.Context, store *types.Store, options ValidateStoreOptions) error {
	// Check that mount is valid
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	// run the check to see if this is a legacy minio deployment and configure a prefix
	// Only checking that the path is mountable
	deployOptions := FileSystemDeployOptions{
		Namespace:        options.KotsadmNamespace,
		IsOpenShift:      k8sutil.IsOpenShift(clientset),
		ForceReset:       false,
		FileSystemConfig: *store.FileSystem.Config,
	}
	_, writable, err := ValidateFileSystemDeployment(ctx, clientset, deployOptions, *options.RegistryConfig)
	if err != nil {
		return errors.Wrap(err, "could not validate lvp file system")
	}
	if !writable {
		return errors.New("the volume path is not writable")
	}
	return nil
}

func validateMinioFileSystem(ctx context.Context, storeFileSystem *types.StoreFileSystem, bucket string, options ValidateStoreOptions) error {
	if options.ValidateUsingAPod {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return errors.Wrap(err, "failed to get k8s clientset")
		}

		podName := fmt.Sprintf("kotsadm-validate-fs-bucket-%d", time.Now().Unix())

		podOptions := kotss3.S3OpsPodOptions{
			PodName:         podName,
			Endpoint:        storeFileSystem.Endpoint,
			BucketName:      bucket,
			AccessKeyID:     storeFileSystem.AccessKeyID,
			SecretAccessKey: storeFileSystem.SecretAccessKey,
			Namespace:       options.KotsadmNamespace,
			IsOpenShift:     k8sutil.IsOpenShift(clientset),
			RegistryConfig:  options.RegistryConfig,
		}

		return kotss3.HeadS3BucketUsingAPod(ctx, clientset, podOptions)
	}

	s3Config := &aws.Config{
		Region:           aws.String(storeFileSystem.Region),
		Endpoint:         aws.String(storeFileSystem.Endpoint),
		DisableSSL:       aws.Bool(true), // TODO: this needs to be configurable
		S3ForcePathStyle: aws.Bool(true),
	}

	if storeFileSystem.AccessKeyID != "" && storeFileSystem.SecretAccessKey != "" {
		s3Config.Credentials = credentials.NewStaticCredentials(storeFileSystem.AccessKeyID, storeFileSystem.SecretAccessKey, "")
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return errors.Wrap(err, "failed to create s3 session")
	}
	s3Client := s3.New(newSession)

	_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		return errors.Wrap(err, "bucket does not exist")
	}

	return nil
}

func Redact(store *types.Store) error {
	if store == nil {
		return nil
	}

	if store.AWS != nil {
		if store.AWS.SecretAccessKey != "" {
			store.AWS.SecretAccessKey = "--- REDACTED ---"
		}
	}

	if store.Google != nil {
		if store.Google.JSONFile != "" {
			store.Google.JSONFile = "--- REDACTED ---"
		}
	}

	if store.Azure != nil {
		if store.Azure.ClientSecret != "" {
			store.Azure.ClientSecret = "--- REDACTED ---"
		}
	}

	if store.Other != nil {
		if store.Other.SecretAccessKey != "" {
			store.Other.SecretAccessKey = "--- REDACTED ---"
		}
	}

	if store.Internal != nil {
		if store.Internal.SecretAccessKey != "" {
			store.Internal.SecretAccessKey = "--- REDACTED ---"
		}
	}

	if store.FileSystem != nil {
		if store.FileSystem.SecretAccessKey != "" {
			store.FileSystem.SecretAccessKey = "--- REDACTED ---"
		}
	}

	return nil
}

func resetRepositories(ctx context.Context, veleroNamespace string) error {
	// velero 1.10+
	if err := resetBackupRepositories(ctx, veleroNamespace); err != nil {
		return errors.Wrap(err, "failed to reset backup repositories")
	}
	// velero < 1.10
	if err := resetResticRepositories(ctx, veleroNamespace); err != nil {
		return errors.Wrap(err, "failed to reset restic repositories")
	}
	return nil
}

func resetBackupRepositories(ctx context.Context, veleroNamespace string) error {
	// BackupRepositories store the previous snapshot location which breaks volume backup when location changes.
	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create velero client")
	}

	var repos velerov1.BackupRepositoryList
	err = veleroClient.List(ctx, &repos, kbclient.InNamespace(veleroNamespace), &kbclient.MatchingLabels{
		"velero.io/storage-location": "default",
	})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to list backuprepositories")
	}
	if kuberneteserrors.IsNotFound(err) {
		return nil
	}

	for _, repo := range repos.Items {
		err := veleroClient.Delete(ctx, &repo)
		if err != nil {
			return errors.Wrapf(err, "failed to delete backuprepository %s", repo.Name)
		}
	}

	return nil
}

func resetResticRepositories(ctx context.Context, veleroNamespace string) error {
	dynamicClient, err := k8sutil.GetDynamicClient()
	if err != nil {
		return errors.Wrap(err, "failed to create dynamic client")
	}

	resticReposClient := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "resticrepositories",
	})

	resticRepos, err := resticReposClient.Namespace(veleroNamespace).List(ctx, metav1.ListOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get resticrepositories")
	}
	if kuberneteserrors.IsNotFound(err) {
		return nil
	}

	for _, repo := range resticRepos.Items {
		err := resticReposClient.Namespace(veleroNamespace).Delete(ctx, repo.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete resticrepository %s", repo.GetName())
		}
	}

	return nil
}

// GetLvpProvider returns the name of the Lvp provider corresponding to the desired filesystem
// configuration
func GetLvpProvider(fsConfig *types.FileSystemConfig) string {
	if fsConfig.HostPath != nil {
		return SnapshotStoreHostPathProvider
	}
	return SnapshotStoreNFSProvider
}

// GetLvpBucket returns the bucket/volume name used for the LVP backup. It includes a hash of the
// Filesystem configuration
func GetLvpBucket(fsConfig *types.FileSystemConfig) (string, error) {
	b, err := json.Marshal(fsConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal filesystem config")
	}
	hash := md5.Sum(b)
	hashId := hex.EncodeToString(hash[:6])
	return fmt.Sprintf("velero-lvp-%s", hashId), nil
}

// WaitForDefaultBslAvailableAndSynced blocks execution until the default backup storage location to display a status as "AVAILABLE"
// and also until the backups in the location are available through the Velero api. There is a timeout of 5 minutes, though the
// default Velero sync time is only 1 minute.
func WaitForDefaultBslAvailableAndSynced(ctx context.Context, veleroNamespace string, start metav1.Time) error {
	timeout := time.After(5 * time.Minute)

	veleroClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create velero client")
	}

	for {
		select {
		case <-timeout:
			return errors.New("timed out waiting for default backup storage location to be available")
		default:
			var bsl velerov1.BackupStorageLocation
			err := veleroClient.Get(ctx, k8stypes.NamespacedName{Namespace: veleroNamespace, Name: DefaultBackupStorageLocationName}, &bsl)
			if err != nil {
				return errors.Wrap(err, "failed to get default backup storage location")
			}

			if bsl.Status.Phase == velerov1.BackupStorageLocationPhaseAvailable && bsl.Status.LastSyncedTime != nil {
				return nil
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func isMinioMigration(clientset kubernetes.Interface, namespace string) bool {
	_, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), SnapshotMigrationArtifactName, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			fmt.Println("Failed to check if Minio migration, defaulting to false:", err)
		}
		return false
	}
	return true
}
