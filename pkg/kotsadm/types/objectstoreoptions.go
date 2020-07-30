package types

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"strconv"
)

const ObjectStoreTypeInternal = "internal"
const ObjectStoreTypeExternal = "external"
const ObjectStoreTypeNone = "none"

// Using an interface to try to avoid potential footguns with defaults management
// and passing around invalid configurations. The only way to get an instance of this
// is to call NewObjectStoreConfig which gives a validated, hydrated object that
// can be used to read/write object store secrets
type ObjectStoreConfig interface {
	Type() string
	LoadSecretData(map[string][]byte) error
	ToSecretData() map[string][]byte
}

var _ ObjectStoreConfig = &objectStoreOptions{}

type objectStoreOptions struct {
	options StorageOptions
}

func (o *objectStoreOptions) Type() string {
	return o.options.ObjectStoreType
}

func (o *objectStoreOptions) LoadSecretData(data map[string][]byte) error {
	return o.options.loadSecretData(data)
}

func (o *objectStoreOptions) ToSecretData() map[string][]byte {
	return o.options.toSecretData()
}

func NewObjectStoreConfig(options StorageOptions) (*objectStoreOptions, error) {
	err := options.validateAndHydrate()
	if err != nil {
		return nil, err
	}
	return &objectStoreOptions{options}, nil
}

func MustGetObjectStoreConfig(options StorageOptions) ObjectStoreConfig {
	config, err := NewObjectStoreConfig(options)
	if err != nil {
		panic(err)
	}
	return config
}

func DefaultObjectStore() ObjectStoreConfig {
	return MustGetObjectStoreConfig(StorageOptions{})
}

type StorageOptions struct {
	ObjectStoreType  string
	AccessKeyID      string
	SecretAccessKey  string
	BucketName       string
	Endpoint         string
	Region           string
	BucketInPath     bool
	SkipEnsureBucket bool

	// Other storage flags for proper validation
	StorageIncludeMinio              bool
	StorageIncludeDockerDistribution bool
	StorageBaseURI                   string
}

func (o *StorageOptions) validateAndHydrate() error {
	// this will probably be initialized in cobra flags but being extra safe
	if o.ObjectStoreType == "" {
		o.ObjectStoreType = ObjectStoreTypeInternal
	}

	if o.ObjectStoreType != ObjectStoreTypeInternal && o.ObjectStoreType != ObjectStoreTypeExternal && o.ObjectStoreType != ObjectStoreTypeNone {
		return errors.Errorf("unsupported object store type: %s", o.ObjectStoreType)
	}

	if o.ObjectStoreType == ObjectStoreTypeNone {
		if o.StorageIncludeMinio {
			return errors.Errorf(`when object store is "none", deploy-minio must not be set`)
		}

		// we could default this here but the logic for defaulting to an internal service is already elsewhere
		// and kots CLI doesn't consume this value directly anyways, just for validation right now, runs after defaults are set
		if o.StorageBaseURI == "" {
			return errors.Errorf(`when object store is "none", storage-base-uri must be set`)
		}
	}

	if o.ObjectStoreType == ObjectStoreTypeInternal {

		if !o.StorageIncludeMinio {
			return errors.Errorf(`when object store is "internal", deploy-minio must be set`)
		}

		if o.AccessKeyID == "" {
			o.AccessKeyID = uuid.New().String()
		}

		if o.SecretAccessKey == "" {
			o.SecretAccessKey = uuid.New().String()
		}

		if o.BucketName == "" {
			o.BucketName = "kotsadm"
		}

		if o.BucketName != "kotsadm" {
			return errors.Errorf(`when object store is "internal", bucket name must be empty`)
		}

		if o.Endpoint == "" {
			o.Endpoint = "http://kotsadm-minio:9000"
		}

		if o.Endpoint != "http://kotsadm-minio:9000" {
			return errors.Errorf(`when object store is "internal", endpoint must be empty`)
		}

		if !o.BucketInPath {
			return errors.Errorf(`when object store is "internal", bucket-in-path must be true`)
		}
	}

	if o.ObjectStoreType == ObjectStoreTypeExternal {
		o.SkipEnsureBucket = true

		if o.AccessKeyID == "" || o.SecretAccessKey == "" || o.BucketName == "" {
			return errors.Errorf(`when object store is "external", each of object-store-access-key-id, object-store-secret-access-key, object-store-bucket-name must be set`)
		}

		if o.Region == "" {
			return errors.Errorf(`when object store is external, must supply a region`)
		}

		// if no endpoint, assume AWS S3
		if o.Endpoint == "" {

			if o.Region == "" {
				return errors.Errorf(`when object store is external and endpoint is not specified, must supply a region`)
			}

			if o.BucketInPath {
				o.Endpoint = fmt.Sprintf("https://s3-%s.amazonaws.com/%s", o.Region, o.BucketName)
			} else {
				o.Endpoint = fmt.Sprintf("https://%s.s3-%s.amazonaws.com", o.BucketName, o.Region)
			}
		}

	}

	return nil
}

func (o *StorageOptions) toSecretData() map[string][]byte {
	skipEnsureBucket := ""
	if o.SkipEnsureBucket {
		skipEnsureBucket = "1"
	}

	return map[string][]byte{
		"type":               []byte(o.ObjectStoreType),
		"accesskey":          []byte(o.AccessKeyID),
		"secretkey":          []byte(o.SecretAccessKey),
		"endpoint":           []byte(o.Endpoint),
		"bucketname":         []byte(o.BucketName),
		"region":             []byte(o.Region),
		"bucket-in-path":     []byte(fmt.Sprintf("%t", o.BucketInPath)),
		"skip-ensure-bucket": []byte(skipEnsureBucket),
	}
}

func (o *StorageOptions) loadSecretData(secretData map[string][]byte) error {

	if accessKey, ok := secretData["accesskey"]; ok {
		o.AccessKeyID = string(accessKey)
	}

	if secretKey, ok := secretData["secretkey"]; ok {
		o.SecretAccessKey = string(secretKey)
	}

	if endpoint, ok := secretData["endpoint"]; ok {
		o.Endpoint = string(endpoint)
	}

	if bucketName, ok := secretData["bucketname"]; ok {
		o.BucketName = string(bucketName)
	}

	if region, ok := secretData["region"]; ok {
		o.Region = string(region)
	}

	if skipEnsureBucket, ok := secretData["skip-ensure-bucket"]; ok {
		o.SkipEnsureBucket = string(skipEnsureBucket) != ""
	}

	if bucketInPathBytes, ok := secretData["bucket-in-path"]; ok {
		bucketInPath, err := strconv.ParseBool(string(bucketInPathBytes))
		if err != nil {
			return errors.Wrap(err, "parse bucket-in-path key of secretData")
		}
		o.BucketInPath = bucketInPath
	}

	// just for fun, let's validate it after loading
	err := o.validateAndHydrate()
	if err != nil {
		return errors.Wrap(err, "validate object store config")
	}

	return nil

}
