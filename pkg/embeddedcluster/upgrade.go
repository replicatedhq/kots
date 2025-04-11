package embeddedcluster

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/archiver/v3"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	embeddedclustertypes "github.com/replicatedhq/embedded-cluster/kinds/types"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	V2MigrationSecretName = "migratev2-secret"
)

// startClusterUpgrade will create a new installation with the provided config.
func startClusterUpgrade(
	ctx context.Context, newcfg embeddedclusterv1beta1.ConfigSpec,
	artifacts *embeddedclusterv1beta1.ArtifactsLocation,
	registrySettings registrytypes.RegistrySettings,
	license *kotsv1beta1.License, versionLabel string,
) error {
	// TODO(upgrade): put a lock here to prevent multiple upgrades at the same time

	kbClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get kubeclient: %w", err)
	}

	k8sClient, err := k8sutil.GetClientset()
	if err != nil {
		return fmt.Errorf("failed to get clientset: %w", err)
	}

	current, err := GetCurrentInstallation(ctx, kbClient)
	if err != nil {
		return fmt.Errorf("failed to get current installation: %w", err)
	}

	newInstall := &embeddedclusterv1beta1.Installation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: embeddedclusterv1beta1.GroupVersion.String(),
			Kind:       "Installation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: time.Now().Format("20060102150405"),
			Labels: map[string]string{
				"replicated.com/disaster-recovery": "ec-install",
			},
		},
		Spec: current.Spec,
	}
	newInstall.Spec.Artifacts = artifacts
	newInstall.Spec.Config = &newcfg
	newInstall.Spec.LicenseInfo = &embeddedclusterv1beta1.LicenseInfo{
		IsDisasterRecoverySupported: license.Spec.IsDisasterRecoverySupported,
		IsMultiNodeEnabled:          license.Spec.IsEmbeddedClusterMultiNodeEnabled,
	}

	log.Printf("Starting cluster upgrade to version %s...", newcfg.Version)

	// We cannot notify the upgrade started until the new install is available
	if err := NotifyUpgradeStarted(ctx, util.ReplicatedAppEndpoint(license), newInstall, current, versionLabel); err != nil {
		logger.Errorf("Failed to notify upgrade started: %v", err)
	}

	err = runClusterUpgrade(ctx, k8sClient, newInstall, registrySettings, license, versionLabel)
	if err != nil {
		if err := NotifyUpgradeFailed(ctx, util.ReplicatedAppEndpoint(license), newInstall, current, err.Error()); err != nil {
			logger.Errorf("Failed to notify upgrade failed: %v", err)
		}
		return fmt.Errorf("run cluster upgrade: %w", err)
	}

	log.Printf("Cluster upgrade to version %s started successfully", newcfg.Version)

	return nil
}

// runClusterUpgrade will download the new embedded cluster operator binary and run the upgrade
// command with the provided installation data. This is needed to get the latest
// embeddedclusterv1beta1 API version. The upgrade command will first upgrade the embedded cluster
// operator, wait for the CRD to be up-to-date, and then apply the installation object.
func runClusterUpgrade(
	ctx context.Context, k8sClient kubernetes.Interface,
	in *embeddedclusterv1beta1.Installation,
	registrySettings registrytypes.RegistrySettings,
	license *kotsv1beta1.License, versionLabel string,
) error {
	var bin string

	if in.Spec.AirGap {
		artifact := in.Spec.Artifacts.AdditionalArtifacts["operator"]
		if artifact == "" {
			return fmt.Errorf("missing operator binary in airgap artifacts")
		}

		b, err := pullUpgradeBinaryFromRegistry(ctx, k8sClient, registrySettings, artifact)
		if err != nil {
			return fmt.Errorf("pull upgrade binary from registry: %w", err)
		}
		bin = b
	} else {
		b, err := downloadUpgradeBinary(ctx, license, versionLabel)
		if err != nil {
			return fmt.Errorf("download upgrade binary: %w", err)
		}
		bin = b
	}
	defer os.RemoveAll(bin)

	err := os.Chmod(bin, 0755)
	if err != nil {
		return fmt.Errorf("chmod upgrade binary: %w", err)
	}

	installationData, err := k8syaml.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal installation: %w", err)
	}

	args := []string{"upgrade"}
	if in.Spec.AirGap {
		// TODO(upgrade): local-artifact-mirror-image should be included in the installation object
		localArtifactMirrorImage, err := getLocalArtifactMirrorImage(ctx, k8sClient, in, registrySettings)
		if err != nil {
			return fmt.Errorf("get local artifact mirror image: %w", err)
		}
		args = append(args, "--local-artifact-mirror-image", localArtifactMirrorImage)
	}
	args = append(args, "--installation", "-")

	log.Printf("Running upgrade command with args %q ...", args)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(string(installationData))

	// create pipe for capturing output
	pr, pw := io.Pipe()
	defer pw.Close()

	// capture stderr separately so we can return it in the error
	var stderr bytes.Buffer
	cmd.Stdout = pw
	cmd.Stderr = io.MultiWriter(pw, &stderr)

	// stream output to logs
	go func() {
		defer pr.Close()
		log.Println("Upgrade command output:")
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			log.Println("  " + scanner.Text())
		}
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run upgrade command: %w: %s", err, stderr.String())
	}

	return nil
}

const (
	// TODO(upgrade): perhaps do not hardcode these
	upgradeBinary         = "operator"
	upgradeBinaryOCIAsset = "operator.tar.gz"
)

func downloadUpgradeBinary(ctx context.Context, license *kotsv1beta1.License, versionLabel string) (string, error) {
	tmpdir, err := os.MkdirTemp("", "embedded-cluster-artifact-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	req, err := newDownloadUpgradeBinaryRequest(ctx, license, versionLabel)
	if err != nil {
		return "", fmt.Errorf("new download upgrade binary request: %w", err)
	}

	log.Printf("Downloading upgrade binary from %s...", req.URL)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	archiveFilepath := filepath.Join(tmpdir, "operator.tar.gz")
	f, err := os.Create(archiveFilepath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.RemoveAll(f.Name())
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", fmt.Errorf("copy response body: %w", err)
	}

	err = unarchive(archiveFilepath, tmpdir)
	if err != nil {
		return "", fmt.Errorf("unarchive: %w", err)
	}

	return filepath.Join(tmpdir, upgradeBinary), nil
}

func newDownloadUpgradeBinaryRequest(ctx context.Context, license *kotsv1beta1.License, versionLabel string) (*http.Request, error) {
	url := fmt.Sprintf("%s/clusterconfig/artifact/operator?versionLabel=%s", util.ReplicatedAppEndpoint(license), url.QueryEscape(versionLabel))
	req, err := util.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.SetBasicAuth(license.Spec.LicenseID, license.Spec.LicenseID)
	req = req.WithContext(ctx)

	return req, nil
}

func pullUpgradeBinaryFromRegistry(
	ctx context.Context, k8sClient kubernetes.Interface,
	registrySettings registrytypes.RegistrySettings,
	repo string,
) (string, error) {
	tmpdir, err := os.MkdirTemp("", "embedded-cluster-artifact-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	log.Printf("Pulling upgrade binary from %s...", repo)

	err = pullFromRegistry(ctx, k8sClient, registrySettings, repo, tmpdir)
	if err != nil {
		return "", fmt.Errorf("pull from registry: %w", err)
	}

	err = unarchive(filepath.Join(tmpdir, upgradeBinaryOCIAsset), tmpdir)
	if err != nil {
		return "", fmt.Errorf("unarchive: %w", err)
	}

	return filepath.Join(tmpdir, upgradeBinary), nil
}

const (
	localArtifactMirrorMetadataKey = "local-artifact-mirror-image"
)

func getLocalArtifactMirrorImage(
	ctx context.Context, k8sClient kubernetes.Interface,
	in *embeddedclusterv1beta1.Installation, registrySettings registrytypes.RegistrySettings,
) (string, error) {
	path, err := pullEmbeddedClusterMetadataFromRegistry(ctx, k8sClient, registrySettings, in.Spec.Artifacts.EmbeddedClusterMetadata)
	if err != nil {
		return "", fmt.Errorf("pull embedded cluster metadata from registry: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open metadata file: %w", err)
	}
	defer f.Close()

	var metadata embeddedclustertypes.ReleaseMetadata
	err = json.NewDecoder(f).Decode(&metadata)
	if err != nil {
		return "", fmt.Errorf("decode metadata: %w", err)
	}

	srcImage, ok := metadata.Artifacts[localArtifactMirrorMetadataKey]
	if !ok {
		return "", fmt.Errorf("missing local artifact mirror image in embedded cluster metadata")
	}

	imageName, err := embeddedRegistryImageName(registrySettings, srcImage)
	if err != nil {
		return "", fmt.Errorf("get image name: %w", err)
	}

	return imageName, nil
}

func pullEmbeddedClusterMetadataFromRegistry(
	ctx context.Context, k8sClient kubernetes.Interface,
	registrySettings registrytypes.RegistrySettings,
	repo string,
) (string, error) {
	tmpdir, err := os.MkdirTemp("", "embedded-cluster-artifact-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	log.Printf("Pulling version metadata from %s...", repo)

	err = pullFromRegistry(ctx, k8sClient, registrySettings, repo, tmpdir)
	if err != nil {
		return "", fmt.Errorf("pull from registry: %w", err)
	}

	return filepath.Join(tmpdir, "version-metadata.json"), nil
}

func pullFromRegistry(
	ctx context.Context, k8sClient kubernetes.Interface,
	registrySettings registrytypes.RegistrySettings,
	srcRepo string, dstDir string,
) error {
	store := credentials.NewMemoryStore()
	err := store.Put(ctx, registrySettings.Hostname, auth.Credential{
		Username: registrySettings.Username,
		Password: registrySettings.Password,
	})
	if err != nil {
		return fmt.Errorf("put credential: %w", err)
	}

	transp, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return fmt.Errorf("default transport is not http.Transport")
	}
	transp = transp.Clone()
	transp.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	opts := pullArtifactOptions{}
	opts.client = &auth.Client{
		Client:     &http.Client{Transport: transp},
		Credential: store.Get,
	}

	err = pullArtifact(ctx, srcRepo, dstDir, opts)
	if err != nil {
		return fmt.Errorf("pull oci artifact: %w", err)
	}

	return nil
}

func unarchive(archiveFilepath string, dstDir string) error {
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
			OverwriteExisting:      true,
		},
	}

	err := tarGz.Unarchive(archiveFilepath, dstDir)
	if err != nil {
		return err
	}

	return nil
}

func embeddedRegistryImageName(registrySettings registrytypes.RegistrySettings, srcImage string) (string, error) {
	destRegistry := dockerregistrytypes.RegistryOptions{
		Endpoint:  registrySettings.Hostname,
		Namespace: registrySettings.Namespace,
		Username:  registrySettings.Username,
		Password:  registrySettings.Password,
	}

	return imageutil.DestECImage(destRegistry, srcImage)
}

func createV2MigrationSecret(ctx context.Context, k8sClient kubernetes.Interface, license kotsv1beta1.License) error {
	encoded, err := k8syaml.Marshal(license)
	if err != nil {
		return fmt.Errorf("encode license: %w", err)
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      V2MigrationSecretName,
			Namespace: "embedded-cluster",
		},
		Data: map[string][]byte{
			"license": encoded,
		},
	}
	_, err = k8sClient.CoreV1().Secrets("embedded-cluster").Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create secret: %w", err)
	}

	return nil
}
