package version

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
	"github.com/mholt/archiver"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	kotss3 "github.com/replicatedhq/kots/kotsadm/pkg/s3"
	"go.uber.org/zap"
)

// CreateAppVersion takes an unarchived app, makes an archive and then uploads it
// to s3 with the appID and sequence specified
func CreateAppVersionArchive(appID string, sequence int64, archivePath string) error {
	paths := []string{
		filepath.Join(archivePath, "upstream"),
		filepath.Join(archivePath, "base"),
		filepath.Join(archivePath, "overlays"),
	}

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.RemoveAll(tmpDir)
	fileToUpload := filepath.Join(tmpDir, "archive.tar.gz")

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Archive(paths, fileToUpload); err != nil {
		return errors.Wrap(err, "failed to create archive")
	}

	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		// KOTS 1.15 and earlier only supported s3 and there was no configuration
		storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
	}

	parsedURI, err := url.Parse(storageBaseURI)
	if err != nil {
		return errors.Wrap(err, "failed to parse storage base uri")
	}

	if parsedURI.Scheme == "docker" {
		return createAppVersionDocker(appID, sequence, fileToUpload, storageBaseURI)
	} else if parsedURI.Scheme == "s3" {
		return createAppVersionS3(appID, sequence, fileToUpload, parsedURI)
	}

	return errors.Errorf("unknown storage base uri scheme: %q", parsedURI.Scheme)
}

func refFromAppVersion(appID string, sequence int64, baseURI string) string {
	baseURI = strings.TrimSuffix(baseURI, "/")

	// docker images don't allow a large charset
	// so this names it registry.host/base/lower(app-id):sequence
	ref := fmt.Sprintf("%s/%s:%d", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(appID), sequence)

	return ref
}

// createAppVersionDocker will push archive in fileToUpload to an image registry that implements the Docker HTTP API V3
func createAppVersionDocker(appID string, sequence int64, fileToUpload string, baseURI string) error {
	ref := refFromAppVersion(appID, sequence, baseURI)

	fileContents, err := ioutil.ReadFile(fileToUpload)
	if err != nil {
		return errors.Wrap(err, "failed to read archive file")
	}

	logger.Debug("pushing app archive to docker registry",
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
	desc := memoryStore.Add(fmt.Sprintf("appversion-%s-%d.tar.gz", appID, sequence), "application/gzip", fileContents)
	pushContents := []ocispec.Descriptor{desc}
	pushedDescriptor, err := oras.Push(context.Background(), resolver, ref, memoryStore, pushContents)
	if err != nil {
		return errors.Wrap(err, "failed to push archive to docker registry")
	}

	logger.Info("pushed app archive to docker registry",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence),
		zap.String("ref", ref),
		zap.String("digest", pushedDescriptor.Digest.String()))

	return nil
}

// createAppVersionS3 is the legacy implementation that will upload the file to an s3 compatible object store
func createAppVersionS3(appID string, sequence int64, fileToUpload string, parsedURI *url.URL) error {
	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(fmt.Sprintf("%s/%d.tar.gz", appID, sequence))

	newSession := awssession.New(kotss3.GetConfig())

	s3Client := s3.New(newSession)

	f, err := os.Open(fileToUpload)
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

// GetAppVersionArchive will fetch the archive and return a string that contains a
// directory name where it's extracted into
func GetAppVersionArchive(appID string, sequence int64) (string, error) {
	logger.Debug("getting app version archive",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence))

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
		return getAppVersionDocker(appID, sequence, tmpDir, storageBaseURI)
	} else if parsedURI.Scheme == "s3" {
		return getAppVersionS3(appID, sequence, tmpDir, parsedURI)
	}

	return "", errors.Errorf("unknown storage base uri scheme: %q", parsedURI.Scheme)
}

func getAppVersionDocker(appID string, sequence int64, outputDir string, baseURI string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return "", errors.Errorf("failed to mk temp dir")
	}
	defer os.RemoveAll(tmpDir)

	fileStore := content.NewFileStore(tmpDir)
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
	ref := refFromAppVersion(appID, sequence, baseURI)

	pulledDescriptor, _, err := oras.Pull(context.Background(), resolver, ref, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return "", errors.Wrap(err, "failed to pull from registry storage")
	}

	logger.Debug("pulled app archive from docker registry",
		zap.String("appID", appID),
		zap.Int64("sequence", sequence),
		zap.String("ref", ref),
		zap.String("digest", pulledDescriptor.Digest.String()))

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(filepath.Join(tmpDir, fmt.Sprintf("appversion-%s-%d.tar.gz", appID, sequence)), outputDir); err != nil {
		return "", errors.Wrap(err, "failed to unarchive")
	}

	return outputDir, nil
}

func getAppVersionS3(appID string, sequence int64, outputDir string, parsedURI *url.URL) (string, error) {
	// Get the archive from object store
	newSession := awssession.New(kotss3.GetConfig())

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(fmt.Sprintf("%s/%d.tar.gz", appID, sequence))

	tmpFile, err := ioutil.TempFile("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	defer tmpFile.Close()
	defer os.RemoveAll(tmpFile.Name())

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(tmpFile,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		return "", errors.Wrap(err, "failed to download file")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(tmpFile.Name(), outputDir); err != nil {
		return "", errors.Wrap(err, "failed to unarchive")
	}

	return outputDir, nil
}

func ExtractArchiveToTempDirectory(archiveFilename string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
			OverwriteExisting:      true,
		},
	}
	if err := tarGz.Unarchive(archiveFilename, tmpDir); err != nil {
		return "", errors.Wrap(err, "failed to unarchive")
	}

	return tmpDir, nil
}
