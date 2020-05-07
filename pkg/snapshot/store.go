package snapshot

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"time"

	storagemgmt "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/snapshot/providers"
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
func UpdateGlobalStore(store *types.Store) (*velerov1.BackupStorageLocation, error) {
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

	kotsadmVeleroBackendStorageLocation, err := findBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	kotsadmVeleroBackendStorageLocation.Spec.Provider = store.Provider

	if kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage == nil {
		kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage = &velerov1.ObjectStorageLocation{}
	}

	kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Bucket = store.Bucket
	kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Prefix = store.Path

	currentSecret, currentSecretErr := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get("cloud-credentials", metav1.GetOptions{})
	if currentSecretErr != nil && !kuberneteserrors.IsNotFound(currentSecretErr) {
		return nil, errors.Wrap(currentSecretErr, "failed to read aws secret")
	}

	if store.AWS != nil {
		logger.Debug("updating aws config in global snapshot storage",
			zap.String("region", store.AWS.Region),
			zap.String("accessKeyId", store.AWS.AccessKeyID),
			zap.Bool("useInstanceRole", store.AWS.UseInstanceRole))

		kotsadmVeleroBackendStorageLocation.Spec.Config["region"] = store.AWS.Region
		// s3Url can be set by Other and conflicts with S3
		delete(kotsadmVeleroBackendStorageLocation.Spec.Config, "s3Url")

		if store.AWS.UseInstanceRole {
			// delete the secret
			if currentSecretErr == nil {
				err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Delete("cloud-credentials", &metav1.DeleteOptions{})
				if err != nil {
					return nil, errors.Wrap(err, "failed to delete aws secret")
				}
			}
		} else {
			awsCfg := ini.Empty()
			section, err := awsCfg.NewSection("default")
			if err != nil {
				return nil, errors.Wrap(err, "failed to create default section in aws creds")
			}
			_, err = section.NewKey("aws_access_key_id", store.AWS.AccessKeyID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create access key")
			}

			_, err = section.NewKey("aws_secret_access_key", store.AWS.SecretAccessKey)
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

			// create or update the secret
			if kuberneteserrors.IsNotFound(currentSecretErr) {
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
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(&toCreate)
				if err != nil {
					return nil, errors.Wrap(err, "failed to create aws secret")
				}
			} else {
				// update
				if currentSecret.Data == nil {
					currentSecret.Data = map[string][]byte{}
				}

				currentSecret.Data["cloud"] = awsCredentials.Bytes()
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(currentSecret)
				if err != nil {
					return nil, errors.Wrap(err, "failed to update aws secret")
				}
			}
		}
	} else if store.Other != nil {
		kotsadmVeleroBackendStorageLocation.Spec.Config["region"] = store.Other.Region
		kotsadmVeleroBackendStorageLocation.Spec.Config["s3Url"] = store.Other.Endpoint

		// create or update the secret
	} else if store.Google != nil {
		if store.Google.UseInstanceRole {
			// delete the secret
			if currentSecretErr == nil {
				err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Delete("cloud-credentials", &metav1.DeleteOptions{})
				if err != nil {
					return nil, errors.Wrap(err, "failed to delete google secret")
				}
			}
		} else {
			// create or update the secret
			if kuberneteserrors.IsNotFound(currentSecretErr) {
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
						"cloud": []byte(store.Google.ServiceAccount),
					},
				}
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(&toCreate)
				if err != nil {
					return nil, errors.Wrap(err, "failed to create aws secret")
				}
			} else {
				// update
				if currentSecret.Data == nil {
					currentSecret.Data = map[string][]byte{}
				}

				currentSecret.Data["cloud"] = []byte(store.Google.ServiceAccount)
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(currentSecret)
				if err != nil {
					return nil, errors.Wrap(err, "failed to update aws secret")
				}
			}
		}
	} else if store.Azure != nil {
		kotsadmVeleroBackendStorageLocation.Spec.Config["resourceGroup"] = store.Azure.ResourceGroup
		kotsadmVeleroBackendStorageLocation.Spec.Config["storageAccount"] = store.Azure.StorageAccount
		kotsadmVeleroBackendStorageLocation.Spec.Config["subscriptionId"] = store.Azure.SubscriptionID

		config := providers.Azure{
			SubscriptionID: store.Azure.SubscriptionID,
			TenantID:       store.Azure.TenantID,
			ClientID:       store.Azure.ClientID,
			ClientSecret:   store.Azure.ClientSecret,
			ResourceGroup:  store.Azure.ResourceGroup,
			CloudName:      store.Azure.CloudName,
		}

		// create or update the secret
		if kuberneteserrors.IsNotFound(currentSecretErr) {
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
					"cloud": providers.RenderAzureConfig(config),
				},
			}
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(&toCreate)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create aws secret")
			}
		} else {
			// update
			if currentSecret.Data == nil {
				currentSecret.Data = map[string][]byte{}
			}

			currentSecret.Data["cloud"] = providers.RenderAzureConfig(config)
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(currentSecret)
			if err != nil {
				return nil, errors.Wrap(err, "failed to update aws secret")
			}
		}
	}

	updated, err := veleroClient.BackupStorageLocations(kotsadmVeleroBackendStorageLocation.Namespace).Update(kotsadmVeleroBackendStorageLocation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update backup storage location")
	}

	return updated, nil
}

// GetGlobalStore will return the global store from kotsadmVeleroBackupStorageLocation
// or will find it, is the param is nil
func GetGlobalStore(kotsadmVeleroBackendStorageLocation *velerov1.BackupStorageLocation) (*types.Store, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	if kotsadmVeleroBackendStorageLocation == nil {
		veleroClient, err := veleroclientv1.NewForConfig(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create velero clientset")
		}

		backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to list backupstoragelocations")
		}

		for _, backupStorageLocation := range backupStorageLocations.Items {
			if backupStorageLocation.Name == "default" {
				kotsadmVeleroBackendStorageLocation = &backupStorageLocation
				break
			}
		}
	}

	if kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage == nil {
		return nil, nil
	}

	prefix := kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Prefix
	// TODO: Remove this if nothing gets broken
	// Copied from the typescript implemention, not sure why this would be set
	// if strings.HasPrefix(prefix, "kotsadm-velero-backend") {
	// 	prefix = prefix[len("kotsadm-velero-backend"):]
	// }

	store := types.Store{
		Provider: kotsadmVeleroBackendStorageLocation.Spec.Provider,
		Bucket:   kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Bucket,
		Path:     prefix,
	}

	switch store.Provider {
	case "aws":
		endpoint, isS3Compatible := kotsadmVeleroBackendStorageLocation.Spec.Config["s3Url"]
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
			if !isS3Compatible {
				store.AWS.UseInstanceRole = true
			}
		} else if err == nil {
			awsCfg, err := ini.Load(awsSecret.Data["cloud"])
			if err != nil {
				return nil, errors.Wrap(err, "failed to load aws credentials")
			}

			for _, section := range awsCfg.Sections() {
				if section.Name() == "default" {
					if isS3Compatible {
						store.Other.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.Other.SecretAccessKey = section.Key("aws_secret_access_key").Value()
					} else {
						store.AWS.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.AWS.SecretAccessKey = section.Key("aws_secret_access_key").Value()
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

		// get the secret
		azureSecret, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get("cloud-credentials", metav1.GetOptions{})
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
			store.Azure.CloudName = "AzurePublicCloud"
		}

		break

	case "gcp":
		// get the secret
		currentSecret, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get("cloud-credentials", metav1.GetOptions{})
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "failed to read google secret")
		}

		serviceAccount := ""
		if err == nil {
			currentServiceAccount, ok := currentSecret.Data["cloud"]
			if ok {
				serviceAccount = string(currentServiceAccount)
			}
		}

		store.Google = &types.StoreGoogle{
			ServiceAccount:  serviceAccount,
			UseInstanceRole: serviceAccount == "",
		}
		break
	}

	return &store, nil
}

func findBackupStoreLocation() (*velerov1.BackupStorageLocation, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list backupstoragelocations")
	}

	for _, backupStorageLocation := range backupStorageLocations.Items {
		if backupStorageLocation.Name == "default" {
			return &backupStorageLocation, nil
		}
	}

	return nil, errors.New("global config not found")
}

func ValidateStore(store *types.Store) error {
	if store.AWS != nil {
		if err := validateAWS(store.AWS, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate AWS configuration")
		}
		return nil
	}

	if store.Azure != nil {
		if err := validateAzure(store.Azure, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate Azure configuration")
		}
		return nil
	}

	if store.Google != nil {
		return nil // TODO: implement
	}

	if store.Other != nil {
		if err := validateOther(store.Other, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate S3-compatible configuration")
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

	if storeAWS.UseInstanceRole {
		s3Config.Credentials = credentials.NewChainCredentials([]credentials.Provider{
			&ec2rolecreds.EC2RoleProvider{
				Client:       ec2metadata.New(session.New()),
				ExpiryWindow: 5 * time.Minute,
			},
		})
	} else {
		s3Config.Credentials = credentials.NewStaticCredentials(storeAWS.AccessKeyID, storeAWS.SecretAccessKey, "")
	}

	newSession := session.New(s3Config)
	s3Client := s3.New(newSession)

	_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		return errors.Wrap(err, "bucket does not exist")
	}

	return nil
}

func validateAzure(storeAzure *types.StoreAzure, bucket string) error {
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

	res, err := storageAccountsClient.ListKeys(context.TODO(), storeAzure.ResourceGroup, storeAzure.StorageAccount)
	if err != nil {
		return errors.Wrap(err, "failed to list account keys")
	}
	if res.Keys == nil || len(*res.Keys) == 0 {
		return errors.New("No storage keys found")
	}

	var storageKey string
	for _, key := range *res.Keys {
		// uppercase both strings for comparison because the ListKeys call returns e.g. "FULL" but
		// the storagemgmt.Full constant in the SDK is defined as "Full".
		if strings.ToUpper(string(key.Permissions)) == strings.ToUpper(string(storagemgmt.Full)) {
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

func validateOther(storeOther *types.StoreOther, bucket string) error {
	s3Config := &aws.Config{
		Region:           aws.String(storeOther.Region),
		Endpoint:         aws.String(storeOther.Endpoint),
		DisableSSL:       aws.Bool(true), // TODO: this needs to be configurable
		S3ForcePathStyle: aws.Bool(true), // TODO: this may need to be configurable
	}

	if storeOther.AccessKeyID != "" && storeOther.SecretAccessKey != "" {
		s3Config.Credentials = credentials.NewStaticCredentials(storeOther.AccessKeyID, storeOther.SecretAccessKey, "")
	}

	newSession := session.New(s3Config)
	s3Client := s3.New(newSession)

	_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
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
		if store.Google.ServiceAccount != "" {
			store.Google.ServiceAccount = "--- REDACTED ---"
		}
	}

	if store.Azure != nil {
		if store.Azure.ClientSecret != "" {
			store.Azure.ClientSecret = "--- REDACTED ---"
		}
	}

	return nil
}
