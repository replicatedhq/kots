package snapshot

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/snapshot/types"
	"github.com/replicatedhq/kots/pkg/kotsadmsnapshot/providers"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	"go.uber.org/zap"
	"google.golang.org/api/option"
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

	kotsadmVeleroBackendStorageLocation, err := FindBackupStoreLocation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find backupstoragelocations")
	}

	kotsadmVeleroBackendStorageLocation.Spec.Provider = store.Provider

	if kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage == nil {
		kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage = &velerov1.ObjectStorageLocation{}
	}

	kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Bucket = store.Bucket
	kotsadmVeleroBackendStorageLocation.Spec.ObjectStorage.Prefix = store.Path

	currentSecret, currentSecretErr := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get(context.TODO(), "cloud-credentials", metav1.GetOptions{})
	if currentSecretErr != nil && !kuberneteserrors.IsNotFound(currentSecretErr) {
		return nil, errors.Wrap(currentSecretErr, "failed to read aws secret")
	}

	if store.AWS != nil {
		logger.Debug("updating aws config in global snapshot storage",
			zap.String("region", store.AWS.Region),
			zap.String("accessKeyId", store.AWS.AccessKeyID),
			zap.Bool("useInstanceRole", store.AWS.UseInstanceRole))

		kotsadmVeleroBackendStorageLocation.Spec.Config = map[string]string{
			"region": store.AWS.Region,
		}

		if store.AWS.UseInstanceRole {
			// delete the secret
			if currentSecretErr == nil {
				err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Delete(context.TODO(), "cloud-credentials", metav1.DeleteOptions{})
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
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(context.TODO(), &toCreate, metav1.CreateOptions{})
				if err != nil {
					return nil, errors.Wrap(err, "failed to create aws secret")
				}
			} else {
				// update
				if currentSecret.Data == nil {
					currentSecret.Data = map[string][]byte{}
				}

				currentSecret.Data["cloud"] = awsCredentials.Bytes()
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(context.TODO(), currentSecret, metav1.UpdateOptions{})
				if err != nil {
					return nil, errors.Wrap(err, "failed to update aws secret")
				}
			}
		}
	} else if store.Other != nil {
		kotsadmVeleroBackendStorageLocation.Spec.Config = map[string]string{
			"region":           store.Other.Region,
			"s3Url":            store.Other.Endpoint,
			"s3ForcePathStyle": "true",
		}

		otherCfg := ini.Empty()
		section, err := otherCfg.NewSection("default")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create default section in other creds")
		}
		_, err = section.NewKey("aws_access_key_id", store.Other.AccessKeyID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create other access key id")
		}

		_, err = section.NewKey("aws_secret_access_key", store.Other.SecretAccessKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create other secret access key")
		}

		var otherCredentials bytes.Buffer
		writer := bufio.NewWriter(&otherCredentials)
		_, err = otherCfg.WriteTo(writer)
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
					"cloud": otherCredentials.Bytes(),
				},
			}
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(context.TODO(), &toCreate, metav1.CreateOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "failed to create other secret")
			}
		} else {
			// update
			if currentSecret.Data == nil {
				currentSecret.Data = map[string][]byte{}
			}

			currentSecret.Data["cloud"] = otherCredentials.Bytes()
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(context.TODO(), currentSecret, metav1.UpdateOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "failed to update other secret")
			}
		}
	} else if store.Internal != nil {
		kotsadmVeleroBackendStorageLocation.Spec.Config = map[string]string{
			"region":           store.Internal.Region,
			"s3Url":            store.Internal.Endpoint,
			"publicUrl":        fmt.Sprintf("http://%s", store.Internal.ObjectStoreClusterIP),
			"s3ForcePathStyle": "true",
		}

		internalCfg := ini.Empty()
		section, err := internalCfg.NewSection("default")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create default section in internal creds")
		}
		_, err = section.NewKey("aws_access_key_id", store.Internal.AccessKeyID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create internal access key id")
		}

		_, err = section.NewKey("aws_secret_access_key", store.Internal.SecretAccessKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create internal secret access key")
		}

		var internalCredentials bytes.Buffer
		writer := bufio.NewWriter(&internalCredentials)
		_, err = internalCfg.WriteTo(writer)
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
					"cloud": internalCredentials.Bytes(),
				},
			}
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(context.TODO(), &toCreate, metav1.CreateOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "failed to create internal secret")
			}
		} else {
			// update
			if currentSecret.Data == nil {
				currentSecret.Data = map[string][]byte{}
			}

			currentSecret.Data["cloud"] = internalCredentials.Bytes()
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(context.TODO(), currentSecret, metav1.UpdateOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "failed to update internal secret")
			}
		}
	} else if store.Google != nil {
		if store.Google.UseInstanceRole {
			kotsadmVeleroBackendStorageLocation.Spec.Config["serviceAccount"] = store.Google.ServiceAccount

			// delete the secret
			if currentSecretErr == nil {
				err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Delete(context.TODO(), "cloud-credentials", metav1.DeleteOptions{})
				if err != nil {
					return nil, errors.Wrap(err, "failed to delete google secret")
				}
			}
		} else {
			delete(kotsadmVeleroBackendStorageLocation.Spec.Config, "serviceAccount")

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
						"cloud": []byte(store.Google.JSONFile),
					},
				}
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(context.TODO(), &toCreate, metav1.CreateOptions{})
				if err != nil {
					return nil, errors.Wrap(err, "failed to create google secret")
				}
			} else {
				// update
				if currentSecret.Data == nil {
					currentSecret.Data = map[string][]byte{}
				}

				currentSecret.Data["cloud"] = []byte(store.Google.JSONFile)
				_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(context.TODO(), currentSecret, metav1.UpdateOptions{})
				if err != nil {
					return nil, errors.Wrap(err, "failed to update google secret")
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
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Create(context.TODO(), &toCreate, metav1.CreateOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "failed to create azure secret")
			}
		} else {
			// update
			if currentSecret.Data == nil {
				currentSecret.Data = map[string][]byte{}
			}

			currentSecret.Data["cloud"] = providers.RenderAzureConfig(config)
			_, err = clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Update(context.TODO(), currentSecret, metav1.UpdateOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "failed to update azure secret")
			}
		}
	}

	updated, err := veleroClient.BackupStorageLocations(kotsadmVeleroBackendStorageLocation.Namespace).Update(context.TODO(), kotsadmVeleroBackendStorageLocation, metav1.UpdateOptions{})
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

		backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(context.TODO(), metav1.ListOptions{})
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
			s3Secret, err := kurl.GetS3Secret()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get s3 secret")
			}
			if s3Secret != nil && string(s3Secret.Data["endpoint"]) == endpoint {
				store.Internal = &types.StoreInternal{
					Region:               kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
					Endpoint:             endpoint,
					ObjectStoreClusterIP: string(s3Secret.Data["object-store-cluster-ip"]),
				}
			} else {
				store.Other = &types.StoreOther{
					Region:   kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
					Endpoint: endpoint,
				}
			}
		} else {
			store.AWS = &types.StoreAWS{
				Region: kotsadmVeleroBackendStorageLocation.Spec.Config["region"],
			}
		}

		awsSecret, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get(context.TODO(), "cloud-credentials", metav1.GetOptions{})
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
					if store.Internal != nil {
						store.Internal.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.Internal.SecretAccessKey = section.Key("aws_secret_access_key").Value()
					} else if store.Other != nil {
						store.Other.AccessKeyID = section.Key("aws_access_key_id").Value()
						store.Other.SecretAccessKey = section.Key("aws_secret_access_key").Value()
					} else if store.AWS != nil {
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
		azureSecret, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get(context.TODO(), "cloud-credentials", metav1.GetOptions{})
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
		currentSecret, err := clientset.CoreV1().Secrets(kotsadmVeleroBackendStorageLocation.Namespace).Get(context.TODO(), "cloud-credentials", metav1.GetOptions{})
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
			ServiceAccount:  kotsadmVeleroBackendStorageLocation.Spec.Config["serviceAccount"],
			JSONFile:        jsonFile,
			UseInstanceRole: jsonFile == "",
		}
		break
	}

	return &store, nil
}

func FindBackupStoreLocation() (*velerov1.BackupStorageLocation, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create velero clientset")
	}

	backupStorageLocations, err := veleroClient.BackupStorageLocations("").List(context.TODO(), metav1.ListOptions{})
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
		if err := validateGCP(store.Google, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate GCP configuration")
		}
		return nil
	}

	if store.Other != nil {
		if err := validateOther(store.Other, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate S3-compatible configuration")
		}
		return nil
	}

	if store.Internal != nil {
		if err := validateInternal(store.Internal, store.Bucket); err != nil {
			return errors.Wrap(err, "failed to validate Internal configuration")
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

func validateInternal(storeInternal *types.StoreInternal, bucket string) error {
	s3Config := &aws.Config{
		Region:           aws.String(storeInternal.Region),
		Endpoint:         aws.String(storeInternal.Endpoint),
		DisableSSL:       aws.Bool(true), // TODO: this needs to be configurable
		S3ForcePathStyle: aws.Bool(true),
	}

	if storeInternal.AccessKeyID != "" && storeInternal.SecretAccessKey != "" {
		s3Config.Credentials = credentials.NewStaticCredentials(storeInternal.AccessKeyID, storeInternal.SecretAccessKey, "")
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

	return nil
}

func ResetResticRepositories() error {
	// ResticRepositories store the previous snapshot location which breaks volume backup when location changes.
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	storageLocation, err := FindBackupStoreLocation()
	if err != nil {
		return errors.Wrap(err, "failed to find backupstoragelocations")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	repos, err := veleroClient.ResticRepositories(storageLocation.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "velero.io/storage-location=default",
	})
	if err != nil {
		return errors.Wrap(err, "failed to list resticrepositories")
	}

	for _, repo := range repos.Items {
		err := veleroClient.ResticRepositories(storageLocation.Namespace).Delete(context.TODO(), repo.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete resticrepository %s", repo.Name)
		}
	}

	return nil
}
