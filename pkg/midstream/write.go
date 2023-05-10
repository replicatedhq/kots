package midstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/disasterrecovery"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream"
	"github.com/replicatedhq/kots/pkg/util"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	secretFilename                           = "secret.yaml"
	patchesFilename                          = "pullsecrets.yaml"
	disasterRecoveryLabelTransformerFileName = "backup-label-transformer.yaml"
)

type WriteOptions struct {
	MidstreamDir       string
	BaseDir            string
	AppSlug            string
	IsGitOps           bool
	IsOpenShift        bool
	Builder            template.Builder
	HTTPProxyEnvValue  string
	HTTPSProxyEnvValue string
	NoProxyEnvValue    string
	UseHelmInstall     map[string]bool
	NewHelmCharts      []*kotsv1beta1.HelmChart
}

func WriteMidstream(writeMidstreamOptions WriteOptions, processImageOptions image.ProcessImageOptions, b *base.Base, license *kotsv1beta1.License, identityConfig *kotsv1beta1.IdentityConfig, upstreamDir string, log *logger.CLILogger) (*Midstream, error) {
	var images []kustomizetypes.Image
	var objects []k8sdoc.K8sDoc
	var pullSecretRegistries []string
	var pullSecretUsername string
	var pullSecretPassword string

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kotskinds from new upstream")
	}

	identitySpec, err := upstream.LoadIdentity(upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load identity")
	}

	// do not fail on being unable to get dockerhub credentials, since they're just used to increase the rate limit
	var dockerHubRegistryCreds registry.Credentials
	dockerhubSecret, _ := registry.GetDockerHubPullSecret(clientset, util.PodNamespace, processImageOptions.Namespace, processImageOptions.AppSlug)
	if dockerhubSecret != nil {
		dockerHubRegistryCreds, _ = registry.GetCredentialsForRegistryFromConfigJSON(dockerhubSecret.Data[".dockerconfigjson"], registry.DockerHubRegistryName)
	}

	if processImageOptions.RewriteImages {
		// A target registry is configured. Rewrite all images and copy them (if necessary) to the configured registry.
		if processImageOptions.RegistrySettings.IsReadOnly {
			log.ActionWithSpinner("Rewriting images")
			io.WriteString(processImageOptions.ReportWriter, "Rewriting images\n")
		} else {
			log.ActionWithSpinner("Copying images")
			io.WriteString(processImageOptions.ReportWriter, "Copying images\n")
		}

		if processImageOptions.AirgapRoot == "" {
			// This is an online installation. Pull and rewrite images from online and copy them (if necessary) to the configured registry.
			rewriteResult, err := RewriteBaseImages(processImageOptions, writeMidstreamOptions.BaseDir, newKotsKinds, license, dockerHubRegistryCreds, log)
			if err != nil {
				return nil, errors.Wrap(err, "failed to rewrite base images")
			}
			images = rewriteResult.Images
			newKotsKinds.Installation.Spec.KnownImages = rewriteResult.CheckedImages
		} else {
			// This is an airgapped installation. Copy and rewrite images from the airgap bundle to the configured registry.
			result, err := ProcessAirgapImages(processImageOptions, newKotsKinds, license, log)
			if err != nil {
				return nil, errors.Wrap(err, "failed to process airgap images")
			}
			images = result.KustomizeImages
			newKotsKinds.Installation.Spec.KnownImages = result.KnownImages
		}

		objects = base.FindObjectsWithImages(b)

		// Use target registry credentials to create image pull secrets for all objects that have images.
		pullSecretRegistries = []string{processImageOptions.RegistrySettings.Hostname}
		pullSecretUsername = processImageOptions.RegistrySettings.Username
		pullSecretPassword = processImageOptions.RegistrySettings.Password
		if pullSecretUsername == "" {
			pullSecretUsername, pullSecretPassword, err = registry.LoadAuthForRegistry(processImageOptions.RegistrySettings.Hostname)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load registry auth for %q", processImageOptions.RegistrySettings.Hostname)
			}
		}
	} else if license != nil {
		// A target registry is NOT configured. Find and rewrite private images to be proxied through proxy.replicated.com
		findResult, err := findPrivateImages(writeMidstreamOptions, b, newKotsKinds, license, dockerHubRegistryCreds)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find private images")
		}
		images = findResult.Images
		newKotsKinds.Installation.Spec.KnownImages = findResult.CheckedImages
		objects = findResult.Docs

		// Use license to create image pull secrets for all objects that have private images.
		pullSecretRegistries = registry.GetRegistryProxyInfo(license, &newKotsKinds.KotsApplication).ToSlice()
		pullSecretUsername = license.Spec.LicenseID
		pullSecretPassword = license.Spec.LicenseID
	}

	// For the newer style charts, create a new secret per chart as helm adds chart specific
	// details to annotations and labels to it.
	namePrefix := processImageOptions.AppSlug
	for _, v := range writeMidstreamOptions.NewHelmCharts {
		if v.Spec.UseHelmInstall && filepath.Base(b.Path) != "." {
			namePrefix = fmt.Sprintf("%s-%s", processImageOptions.AppSlug, filepath.Base(b.Path))
			break
		}
	}
	pullSecrets, err := registry.PullSecretForRegistries(
		pullSecretRegistries,
		pullSecretUsername,
		pullSecretPassword,
		processImageOptions.Namespace,
		namePrefix,
	)
	if err != nil {
		return nil, errors.Wrap(err, "create pull secret")
	}
	pullSecrets.DockerHubSecret = dockerhubSecret

	if err := upstream.SaveInstallation(&newKotsKinds.Installation, upstreamDir); err != nil {
		return nil, errors.Wrap(err, "failed to save installation")
	}

	m, err := CreateMidstream(b, images, objects, &pullSecrets, identitySpec, identityConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create midstream")
	}

	if err := m.Write(writeMidstreamOptions); err != nil {
		return nil, errors.Wrap(err, "failed to write common midstream")
	}

	return m, nil
}

// RewriteBaseImages Will rewrite images found in base and copy them (if necessary) to the configured registry.
func RewriteBaseImages(options image.ProcessImageOptions, baseDir string, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, dockerHubRegistryCreds registry.Credentials, log *logger.CLILogger) (*base.RewriteImagesResult, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.KotsApplication)

	rewriteImageOptions := base.RewriteImageOptions{
		BaseDir: baseDir,
		Log:     log,
		SourceRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
		DockerHubRegistry: dockerregistrytypes.RegistryOptions{
			Username: dockerHubRegistryCreds.Username,
			Password: dockerHubRegistryCreds.Password,
		},
		DestRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:  options.RegistrySettings.Hostname,
			Namespace: options.RegistrySettings.Namespace,
			Username:  options.RegistrySettings.Username,
			Password:  options.RegistrySettings.Password,
		},
		ReportWriter: options.ReportWriter,
		KotsKinds:    kotsKinds,
		IsAirgap:     options.IsAirgap,
		CopyImages:   options.CopyImages,
	}
	if license != nil {
		rewriteImageOptions.AppSlug = license.Spec.AppSlug
		rewriteImageOptions.SourceRegistry.Username = license.Spec.LicenseID
		rewriteImageOptions.SourceRegistry.Password = license.Spec.LicenseID
	}

	rewriteResult, err := base.RewriteImages(rewriteImageOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to rewrite images")
	}

	return rewriteResult, nil
}

// processAirgapImages Will rewrite images found in the airgap bundle/airgap root and copy them (if necessary) to the configured registry.
func ProcessAirgapImages(options image.ProcessImageOptions, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, log *logger.CLILogger) (*upstream.ProcessAirgapImagesResult, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.KotsApplication)

	processAirgapImageOptions := upstream.ProcessAirgapImagesOptions{
		RootDir:      options.RootDir,
		AirgapRoot:   options.AirgapRoot,
		AirgapBundle: options.AirgapBundle,
		CreateAppDir: options.CreateAppDir,
		PushImages:   !options.RegistrySettings.IsReadOnly && options.PushImages,
		Log:          log,
		ReplicatedRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
		ReportWriter: options.ReportWriter,
		DestinationRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:  options.RegistrySettings.Hostname,
			Namespace: options.RegistrySettings.Namespace,
			Username:  options.RegistrySettings.Username,
			Password:  options.RegistrySettings.Password,
		},
	}
	if license != nil {
		processAirgapImageOptions.ReplicatedRegistry.Username = license.Spec.LicenseID
		processAirgapImageOptions.ReplicatedRegistry.Password = license.Spec.LicenseID
	}

	imagesData, err := ioutil.ReadFile(filepath.Join(options.AirgapRoot, "images.json"))
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "failed to load images file")
	}
	if err == nil {
		var images []kustomizetypes.Image
		err := json.Unmarshal(imagesData, &images)
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "failed to unmarshal images data")
		}
		processAirgapImageOptions.UseKnownImages = true
		processAirgapImageOptions.KnownImages = images
	}

	result, err := upstream.ProcessAirgapImages(processAirgapImageOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process airgap images")
	}

	return result, nil
}

// findPrivateImages Finds and rewrites private images to be proxied through proxy.replicated.com
func findPrivateImages(writeMidstreamOptions WriteOptions, b *base.Base, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, dockerHubRegistryCreds registry.Credentials) (*base.FindPrivateImagesResult, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.KotsApplication)
	allPrivate := kotsKinds.KotsApplication.Spec.ProxyPublicImages

	findPrivateImagesOptions := base.FindPrivateImagesOptions{
		BaseDir: writeMidstreamOptions.BaseDir,
		AppSlug: license.Spec.AppSlug,
		ReplicatedRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:         replicatedRegistryInfo.Registry,
			ProxyEndpoint:    replicatedRegistryInfo.Proxy,
			UpstreamEndpoint: replicatedRegistryInfo.Upstream,
		},
		DockerHubRegistry: dockerregistrytypes.RegistryOptions{
			Username: dockerHubRegistryCreds.Username,
			Password: dockerHubRegistryCreds.Password,
		},
		Installation:     &kotsKinds.Installation,
		AllImagesPrivate: allPrivate,
		HelmChartPath:    b.Path,
		UseHelmInstall:   writeMidstreamOptions.UseHelmInstall,
		KotsKindsImages:  kotsutil.GetImagesFromKotsKinds(kotsKinds),
	}
	findResult, err := base.FindPrivateImages(findPrivateImagesOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find private images")
	}

	return findResult, nil
}

func (m *Midstream) Write(options WriteOptions) error {
	if err := os.MkdirAll(options.MidstreamDir, 0744); err != nil {
		return errors.Wrap(err, "failed to mkdir")
	}

	existingKustomization, err := m.getExistingKustomization(options)
	if err != nil {
		return errors.Wrap(err, "get existing kustomization")
	}

	secretFilename, err := m.writePullSecret(options)
	if err != nil {
		return errors.Wrap(err, "failed to write secret")
	}

	if secretFilename != "" {
		m.Kustomization.Resources = append(m.Kustomization.Resources, secretFilename)
	}

	identityBase, err := m.writeIdentityService(context.TODO(), options)
	if err != nil {
		return errors.Wrap(err, "failed to write identity service")
	}

	if identityBase != "" {
		m.Kustomization.Resources = append(m.Kustomization.Resources, identityBase)
	}

	if err := m.writeObjectsWithPullSecret(options); err != nil {
		return errors.Wrap(err, "failed to write patches")
	}

	// transformers
	drLabelTransformerFilename, err := m.writeDisasterRecoveryLabelTransformer(options)
	if err != nil {
		return errors.Wrap(err, "failed to write disaster recovery label transformer")
	}
	m.Kustomization.Transformers = append(m.Kustomization.Transformers, drLabelTransformerFilename)

	// annotations
	if m.Kustomization.CommonAnnotations == nil {
		m.Kustomization.CommonAnnotations = make(map[string]string)
	}
	m.Kustomization.CommonAnnotations["kots.io/app-slug"] = options.AppSlug

	if existingKustomization != nil {
		// Note that this function does nothing on the initial install
		// if the user is not presented with the config screen.
		m.mergeKustomization(options, *existingKustomization)
	}

	if err := m.writeKustomization(options); err != nil {
		return errors.Wrap(err, "failed to write kustomization")
	}

	return nil
}

func (m *Midstream) getExistingKustomization(options WriteOptions) (*kustomizetypes.Kustomization, error) {
	kustomizationFilename := filepath.Join(options.MidstreamDir, "kustomization.yaml")

	_, err := os.Stat(kustomizationFilename)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "stat existing kustomization")
	}

	k, err := k8sutil.ReadKustomizationFromFile(kustomizationFilename)
	if err != nil {
		return nil, errors.Wrap(err, "load existing kustomization")
	}
	return k, nil
}

func (m *Midstream) mergeKustomization(options WriteOptions, existing kustomizetypes.Kustomization) {
	existing.PatchesStrategicMerge = removeFromPatches(existing.PatchesStrategicMerge, patchesFilename)
	m.Kustomization.PatchesStrategicMerge = uniquePatches(existing.PatchesStrategicMerge, m.Kustomization.PatchesStrategicMerge)

	m.Kustomization.Resources = uniqueStrings(existing.Resources, m.Kustomization.Resources)

	m.Kustomization.Transformers = uniqueStrings(existing.Transformers, m.Kustomization.Transformers)

	// annotations
	if existing.CommonAnnotations == nil {
		existing.CommonAnnotations = make(map[string]string)
	}
	delete(existing.CommonAnnotations, "kots.io/app-sequence")
	m.Kustomization.CommonAnnotations = mergeMaps(m.Kustomization.CommonAnnotations, existing.CommonAnnotations)
}

func removeFromPatches(patches []kustomizetypes.PatchStrategicMerge, filename string) []kustomizetypes.PatchStrategicMerge {
	newPatches := []kustomizetypes.PatchStrategicMerge{}
	for _, patch := range patches {
		if string(patch) != filename {
			newPatches = append(newPatches, patch)
		}
	}
	return newPatches
}

func mergeMaps(new map[string]string, existing map[string]string) map[string]string {
	merged := existing
	if merged == nil {
		merged = make(map[string]string)
	}
	for key, value := range new {
		merged[key] = value
	}
	return merged
}

func (m *Midstream) writeKustomization(options WriteOptions) error {
	relativeBaseDir, err := filepath.Rel(options.MidstreamDir, options.BaseDir)
	if err != nil {
		return errors.Wrap(err, "failed to determine relative path for base from midstream")
	}

	kustomizationFilename := filepath.Join(options.MidstreamDir, "kustomization.yaml")

	m.Kustomization.Bases = []string{
		relativeBaseDir,
	}

	if err := k8sutil.WriteKustomizationToFile(*m.Kustomization, kustomizationFilename); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}

func (m *Midstream) writeDisasterRecoveryLabelTransformer(options WriteOptions) (string, error) {
	additionalLabels := map[string]string{
		"kots.io/app-slug": options.AppSlug,
	}
	drLabelTransformerYAML, err := disasterrecovery.GetLabelTransformerYAML(additionalLabels)
	if err != nil {
		return "", errors.Wrap(err, "failed to get disaster recovery label transformer yaml")
	}

	absFilename := filepath.Join(options.MidstreamDir, disasterRecoveryLabelTransformerFileName)

	if err := ioutil.WriteFile(absFilename, drLabelTransformerYAML, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write disaster recovery label transformer yaml file")
	}

	return disasterRecoveryLabelTransformerFileName, nil
}

func (m *Midstream) writePullSecret(options WriteOptions) (string, error) {
	var secretBytes []byte
	if m.AppPullSecret != nil {
		b, err := k8syaml.Marshal(m.AppPullSecret)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal app pull secret")
		}
		secretBytes = b
	}

	if m.AdminConsolePullSecret != nil {
		if secretBytes != nil {
			secretBytes = append(secretBytes, []byte("\n---\n")...)
		}

		b, err := k8syaml.Marshal(m.AdminConsolePullSecret)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal kots pull secret")
		}
		secretBytes = append(secretBytes, b...)
	}

	if m.DockerHubPullSecret != nil {
		if secretBytes != nil {
			secretBytes = append(secretBytes, []byte("\n---\n")...)
		}

		b, err := k8syaml.Marshal(m.DockerHubPullSecret)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal kots pull secret")
		}
		secretBytes = append(secretBytes, b...)
	}

	if secretBytes == nil {
		return "", nil
	}

	absFilename := filepath.Join(options.MidstreamDir, secretFilename)
	if err := ioutil.WriteFile(absFilename, secretBytes, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write pull secret file")
	}

	return secretFilename, nil
}

func (m *Midstream) writeObjectsWithPullSecret(options WriteOptions) error {
	filename := filepath.Join(options.MidstreamDir, patchesFilename)
	if len(m.DocForPatches) == 0 {
		err := os.Remove(filename)
		if err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to delete pull secret patches")
		}

		return nil
	}

	f, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "failed to create resources file")
	}
	defer f.Close()

	secrets := []*corev1.Secret{}
	if m.AppPullSecret != nil {
		secrets = append(secrets, m.AppPullSecret)
	}
	if m.DockerHubPullSecret != nil {
		secrets = append(secrets, m.DockerHubPullSecret)
	}

	for _, o := range m.DocForPatches {
		for _, secret := range secrets {
			withPullSecret := o.PatchWithPullSecret(secret)
			if withPullSecret == nil {
				continue
			}

			b, err := yaml.Marshal(withPullSecret)
			if err != nil {
				return errors.Wrap(err, "failed to marshal object")
			}

			if _, err := f.Write([]byte("---\n")); err != nil {
				return errors.Wrap(err, "failed to write doc separator")
			}
			if _, err := f.Write(b); err != nil {
				return errors.Wrap(err, "failed to write object")
			}
		}
	}

	m.Kustomization.PatchesStrategicMerge = append(m.Kustomization.PatchesStrategicMerge, patchesFilename)

	return nil
}

func EnsureDisasterRecoveryLabelTransformer(archiveDir string, additionalLabels map[string]string) error {
	labelTransformerExists := false

	dirPath := filepath.Join(archiveDir, "overlays", "midstream")

	// TODO (ch35027): this will not work with multiple kustomization files
	k, err := k8sutil.ReadKustomizationFromFile(filepath.Join(dirPath, "kustomization.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read kustomization file from midstream")
	}

	for _, transformer := range k.Transformers {
		if transformer == disasterRecoveryLabelTransformerFileName {
			labelTransformerExists = true
			break
		}
	}

	if !labelTransformerExists {
		drLabelTransformerYAML, err := disasterrecovery.GetLabelTransformerYAML(additionalLabels)
		if err != nil {
			return errors.Wrap(err, "failed to get disaster recovery label transformer yaml")
		}

		absFilename := filepath.Join(dirPath, disasterRecoveryLabelTransformerFileName)

		if err := ioutil.WriteFile(absFilename, drLabelTransformerYAML, 0644); err != nil {
			return errors.Wrap(err, "failed to write disaster recovery label transformer yaml file")
		}

		k.Transformers = append(k.Transformers, disasterRecoveryLabelTransformerFileName)

		if err := k8sutil.WriteKustomizationToFile(*k, filepath.Join(dirPath, "kustomization.yaml")); err != nil {
			return errors.Wrap(err, "failed to write kustomization file to midstream")
		}
	}

	return nil
}
