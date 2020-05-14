package supportbundle

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	kotss3 "github.com/replicatedhq/kots/kotsadm/pkg/s3"
	"go.uber.org/zap"
)

func refFromBundleID(bundleID string, baseURI string) string {
	baseURI = strings.TrimSuffix(baseURI, "/")

	// docker images don't allow a large charset
	// so this names it registry.host/base/supportbundle:{bundle-id}
	return fmt.Sprintf("%s/supportbundle:%s", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(bundleID))
}

func uploadBundleToDocker(bundleID string, archivePath string, baseURI string) error {
	fileContents, err := ioutil.ReadFile(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to read archive file")
	}

	ref := refFromBundleID(bundleID, baseURI)

	logger.Debug("pushing support bundle to docker registry",
		zap.String("ref", ref))

	options := docker.ResolverOptions{}

	registryHosts := func(host string) ([]docker.RegistryHost, error) {
		registryHost := docker.RegistryHost{
			Client:       http.DefaultClient,
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityPush,
		}

		if os.Getenv("STORAGE_BASEURI_PLAINHTTP") == "true" {
			registryHost.Scheme = "http"
		}

		return []docker.RegistryHost{
			registryHost,
		}, nil
	}

	options.Hosts = registryHosts

	resolver := docker.NewResolver(options)

	memoryStore := content.NewMemoryStore()
	desc := memoryStore.Add("supportbundle.tar.gz", "application/gzip", fileContents)
	pushContents := []ocispec.Descriptor{desc}
	pushedDescriptor, err := oras.Push(context.Background(), resolver, ref, memoryStore, pushContents)
	if err != nil {
		return errors.Wrap(err, "failed to push archive to docker registry")
	}

	logger.Info("pushed support bundle to docker registry",
		zap.String("bundleID", bundleID),
		zap.String("ref", ref),
		zap.String("digest", pushedDescriptor.Digest.String()))

	return nil
}

func uploadBundleToS3(id string, archivePath string) error {
	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(filepath.Join("supportbundles", id, "supportbundle.tar.gz"))

	newSession := awssession.New(kotss3.GetConfig())

	s3Client := s3.New(newSession)

	f, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open archive file")
	}

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Body:   f,
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return errors.Wrap(err, "failed to upload to s3")
	}

	return nil
}

// GetSupportBundle will fetch the bundle archive and return a path to where it
// is stored. The caller is responsible for deleting.
func GetSupportBundle(bundleID string) (string, error) {
	logger.Debug("getting support bundle",
		zap.String("bundleID", bundleID))

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		// KOTS 1.15 and earlier only supported s3 and there was no configuration
		storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
	}

	parsedURI, err := url.Parse(storageBaseURI)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse storage base uri")
	}

	if parsedURI.Scheme == "docker" {
		return getBundleFromDocker(bundleID, tmpDir, storageBaseURI)
	} else if parsedURI.Scheme == "s3" {
		return getBundleFromS3(bundleID, tmpDir, parsedURI)
	}

	return "", errors.Errorf("unknown storage base uri scheme: %q", parsedURI.Scheme)
}

func getBundleFromDocker(bundleID string, outputDir string, baseURI string) (string, error) {
	fileStore := content.NewFileStore(outputDir)
	defer fileStore.Close()

	allowedMediaTypes := []string{"application/gzip"}

	options := docker.ResolverOptions{}

	registryHosts := func(host string) ([]docker.RegistryHost, error) {
		registryHost := docker.RegistryHost{
			Client:       http.DefaultClient,
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityResolve | docker.HostCapabilityPull,
		}

		if os.Getenv("STORAGE_BASEURI_PLAINHTTP") == "true" {
			registryHost.Scheme = "http"
		}

		return []docker.RegistryHost{
			registryHost,
		}, nil
	}

	options.Hosts = registryHosts

	resolver := docker.NewResolver(options)
	ref := refFromBundleID(bundleID, baseURI)
	pulledDescriptor, _, err := oras.Pull(context.Background(), resolver, ref, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return "", errors.Wrap(err, "failed to pull from registry storage")
	}

	logger.Debug("pulled support bundle from docker registry",
		zap.String("bundleID", bundleID),
		zap.String("ref", ref),
		zap.String("digest", pulledDescriptor.Digest.String()))

	return filepath.Join(outputDir, "supportbundle.tar.gz"), nil
}

func getBundleFromS3(bundleID string, outputDir string, parsedURI *url.URL) (string, error) {
	newSession := awssession.New(kotss3.GetConfig())

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(fmt.Sprintf("supportbundles/%s/supportbundle.tar.gz", bundleID))

	outputFile, err := os.Create(filepath.Join(outputDir, "supportbundle.tar.gz"))
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}
	defer outputFile.Close()

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(outputFile,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		return "", errors.Wrap(err, "failed to download file")
	}

	return filepath.Join(outputDir, "supportbundle.tar.gz"), nil
}
