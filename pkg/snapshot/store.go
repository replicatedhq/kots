package snapshot

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/snapshot/types"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	"go.uber.org/zap"
	"gopkg.in/ini.v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// UpdateGlobalStore will update the in-cluster storage with exactly what's in the store param
func UpdateGlobalStore(store *types.Store) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list backupstoragelocations")
	}

	var kotsadmVeleroBackendStorageLocation *velerov1.BackupStorageLocation

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "kotsadm-velero-backend" {
			kotsadmVeleroBackendStorageLocation = &backupStorageLocation
		}
	}

	if kotsadmVeleroBackendStorageLocation == nil {
		return errors.New("global config not found")
	}

	kotsadmVeleroBackendStorageLocation.Spec.Provider = store.Provider

	if kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage == nil {
		kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage = &velerov1.ObjectStorageLocation{}
	}

	kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Bucket = store.Bucket
	kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Prefix = store.Path

	if store.AWS != nil {
		logger.Debug("updating aws config in global snapshot storage",
			zap.String("region", store.AWS.Region),
			zap.Bool("useInstanceRole", store.AWS.UseInstanceRole))

		kotsadmVeleroBackendStorageLocation.Spec.Config["region"] = store.AWS.Region

		awsSecret, awsSecretErr := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get("cloud-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to read aws secret")
		}

		if store.AWS.UseInstanceRole {
			// delete the secret
			if awsSecretErr == nil {
				err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Delete("cloud-credentials", &metav1.DeleteOptions{})
				if err != nil {
					return errors.Wrap(err, "failed to delete aws secret")
				}
			}
		} else {
			awsCfg := ini.Empty()
			section, err := awsCfg.NewSection("default")
			if err != nil {
				return errors.Wrap(err, "failed to create default section in aws creds")
			}
			_, err = section.NewKey("aws_access_key_id", store.AWS.AccessKeyID)
			if err != nil {
				return errors.Wrap(err, "failed to create access key")
			}
			_, err = section.NewKey("aws_secret_access_key", store.AWS.SecretAccessKey)
			if err != nil {
				return errors.Wrap(err, "failed to create secret access key")
			}

			var awsCredentials bytes.Buffer
			writer := bufio.NewWriter(&awsCredentials)
			_, err = awsCfg.WriteTo(writer)
			if err != nil {
				return errors.Wrap(err, "failed to write ini")
			}

			// create or update the secret
			if kuberneteserrors.IsNotFound(awsSecretErr) {
				// create
				toCreate := corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cloud-credentials",
						Namespace: kotsadmVeleroBackendStorageLocation.Namespace,
					},
					Data: map[string][]byte{
						"cloud": awsCredentials.Bytes(),
					},
				}
				_, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(&toCreate)
				if err != nil {
					return errors.Wrap(err, "failed to create aws secret")
				}
			} else {
				// update
				if awsSecret.Data == nil {
					awsSecret.Data = map[string][]byte{}
				}

				awsSecret.Data["cloud"] = awsCredentials.Bytes()
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(awsSecret)
				if err != nil {
					return errors.Wrap(err, "failed to update aws secret")
				}
			}
		}
	} else if store.Other != nil {
		kotsadmVeleroBackendStorageLocation.Spec.Config["region"] = store.Other.Region
		kotsadmVeleroBackendStorageLocation.Spec.Config["s3url"] = store.Other.Endpoint

		// create or update the secret
	} else if store.Google != nil {
		if !store.Google.UseInstanceRole {
			// delete the secret
		} else {
			// create or update the secret
		}
	} else if store.Azure != nil {
		kotsadmVeleroBackendStorageLocation.Spec.Config["resourceGroup"] = store.Azure.ResourceGroup
		kotsadmVeleroBackendStorageLocation.Spec.Config["storageAccount"] = store.Azure.StorageAccount
		kotsadmVeleroBackendStorageLocation.Spec.Config["subscriptionId"] = store.Azure.SubscriptionID
		kotsadmVeleroBackendStorageLocation.Spec.Config["cloudName"] = store.Azure.CloudName

		// create or update the secret
	}
	return nil
}

func GetGlobalStore() (*types.Store, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list backupstoragelocations")
	}

	var kotsadmVeleroBackendStorageLocation *velerov1.BackupStorageLocation
	var defaultBackendStorageLocation *velerov1.BackupStorageLocation

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "kotsadm-velero-backend" {
			kotsadmVeleroBackendStorageLocation = &backupStorageLocation
		} else if backupStorageLocation.Name == "default" {
			defaultBackendStorageLocation = &backupStorageLocation
		}
	}

	if kotsadmVeleroBackendStorageLocation == nil && defaultBackendStorageLocation != nil {
		// copy it to ours... so we can make edits without changing the entire cluster config
		toCreate := defaultBackendStorageLocation.DeepCopy()
		toCreate.Name = "kotsadm-velero-backend"
		toCreate.ResourceVersion = ""

		created, err := veleroClient.BackupStorageLocations(toCreate.Namespace).Create(toCreate)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create kotsadm storage location")
		}

		kotsadmVeleroBackendStorageLocation = created
	}

	if kotsadmVeleroBackendStorageLocation == nil {
		return nil, nil // it's not configured
	}

	if kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage == nil {
		return nil, nil
	}

	prefix := kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Prefix
	// Copied from the typescript implemention, not sure why this would be set
	if strings.HasPrefix(prefix, "kotsadm-velero-backend") {
		prefix = prefix[len("kotsadm-velero-backend"):]
	}

	store := types.Store{
		Provider: kotsadmVeleroBackendStorageLocation.Spec.Provider,
		Bucket:   kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Bucket,
		Path:     prefix,
	}

	switch store.Provider {
	case "aws":
		endpoint, isS3Compatible := kotsadmVeleroBackendStorageLocation.Spec.Config["s3url"]
		if isS3Compatible {
			store.Other = &types.StoreOther{
				Region:   kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
				Endpoint: endpoint,
			}
		} else {
			store.AWS = &types.StoreAWS{
				Region: kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
			}
		}

		awsSecret, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get("cloud-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to read aws secret")
		}

		if kuberneteserrors.IsNotFound(err) {
			store.AWS.UseInstanceRole = true
		} else if err == nil {
			awsCfg, err := ini.Load(awsSecret.Data["cloud"])
			if err != nil {
				return nil, errors.Wrap(err, "failed to load aws credentials")
			}

			for _, section := range awsCfg.Sections() {
				if section.Name() == "default" {
					if isS3Compatible {
						store.Other.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.Other.SecretAccessKey = "--- REDACTED ---"
					} else {
						store.AWS.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.AWS.SecretAccessKey = "--- REDACTED ---"
					}
				}
			}
		}
		break

	case "azure":
		// TODO validate these keys in a real azure account
		store.Azure = &types.StoreAzure{
			ResourceGroup:  kotsadmVeleroBackendStorageLocation.Spec.Config["resourceGroup"],
			StorageAccount: kotsadmVeleroBackendStorageLocation.Spec.Config["storageAccount"],
			SubscriptionID: kotsadmVeleroBackendStorageLocation.Spec.Config["subscriptionId"],
		}

		cloudName, ok := kotsadmVeleroBackendStorageLocation.Spec.Config["cloudName"]
		if ok {
			store.Azure.CloudName = cloudName
		} else {
			store.Azure.CloudName = "AzurePublicCloud"
		}

		// get the secret
		azureSecret, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get("azure-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to read azure secret")
		}

		if err == nil {
			tenantID, ok := azureSecret.Data["tenantId"]
			if ok {
				store.Azure.TenantID = string(tenantID)
			}
			clientID, ok := azureSecret.Data["clientId"]
			if ok {
				store.Azure.ClientID = string(clientID)
			}
			_, ok = azureSecret.Data["clientSecret"]
			if ok {
				store.Azure.ClientSecret = "--- REDACTED ---"
			}
		}
		break

	case "gcp":
		// get the secret
		_, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get("google-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to read google secret")
		}

		serviceAccount := ""
		if err == nil {
			serviceAccount = "--- REDACTED ---"
		}

		store.Google = &types.StoreGoogle{
			ServiceAccount: serviceAccount,
		}
		break
	}

	return &store, nil
}
