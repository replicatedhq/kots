package preflight

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/installers"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotstypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/replicatedhq/kots/pkg/registry"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/render/helper"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	kurlv1beta1 "github.com/replicatedhq/kurlkinds/pkg/apis/cluster/v1beta1"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootcollect "github.com/replicatedhq/troubleshoot/pkg/collect"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

const (
	SpecDataKey = "preflight-spec"
)

func Run(appID string, appSlug string, sequence int64, isAirgap bool, archiveDir string) error {
	renderedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(filepath.Join(archiveDir, "upstream"))
	if err != nil {
		return errors.Wrap(err, "failed to load rendered kots kinds")
	}

	status, err := store.GetStore().GetDownstreamVersionStatus(appID, sequence)
	if err != nil {
		return errors.Wrapf(err, "failed to check downstream version %d status", sequence)
	}

	// preflights should not run until config is finished
	if status == storetypes.VersionPendingConfig {
		logger.Debug("not running preflights for app that is pending required configuration",
			zap.String("appID", appID),
			zap.Int64("sequence", sequence))
		return nil
	}

	var ignoreRBAC bool
	var registrySettings registrytypes.RegistrySettings
	var preflight *troubleshootv1beta2.Preflight

	ignoreRBAC, err = store.GetStore().GetIgnoreRBACErrors(appID, sequence)
	if err != nil {
		return errors.Wrap(err, "failed to get ignore rbac flag")
	}

	registrySettings, err = store.GetStore().GetRegistryDetailsForApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get registry settings for app")
	}

	tsKinds, err := kotsutil.LoadTSKindsFromPath(filepath.Join(archiveDir, "rendered"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to load troubleshoot kinds from path: %s", filepath.Join(archiveDir, "rendered")))
	}

	runPreflights := false
	if tsKinds.PreflightsV1Beta2 != nil {

		for _, v := range tsKinds.PreflightsV1Beta2 {
			preflight = troubleshootpreflight.ConcatPreflightSpec(preflight, &v)
		}

		injectDefaultPreflights(preflight, renderedKotsKinds, registrySettings)

		numAnalyzers := 0
		for _, analyzer := range preflight.Spec.Analyzers {
			exclude := troubleshootanalyze.GetExcludeFlag(analyzer).BoolOrDefaultFalse()
			if !exclude {
				numAnalyzers += 1
			}
		}
		runPreflights = numAnalyzers > 0
	} else if renderedKotsKinds.Preflight != nil {
		// render the preflight file
		// we need to convert to bytes first, so that we can reuse the renderfile function
		renderedMarshalledPreflights, err := renderedKotsKinds.Marshal("troubleshoot.replicated.com", "v1beta1", "Preflight")
		if err != nil {
			return errors.Wrap(err, "failed to marshal rendered preflight")
		}

		renderedPreflight, err := render.RenderFile(renderedKotsKinds, registrySettings, appSlug, sequence, isAirgap, util.PodNamespace, []byte(renderedMarshalledPreflights))
		if err != nil {
			return errors.Wrap(err, "failed to render preflights")
		}
		preflight, err = kotsutil.LoadPreflightFromContents(renderedPreflight)
		if err != nil {
			return errors.Wrap(err, "failed to load rendered preflight")
		}

		injectDefaultPreflights(preflight, renderedKotsKinds, registrySettings)

		numAnalyzers := 0
		for _, analyzer := range preflight.Spec.Analyzers {
			exclude := troubleshootanalyze.GetExcludeFlag(analyzer).BoolOrDefaultFalse()
			if !exclude {
				numAnalyzers += 1
			}
		}
		runPreflights = numAnalyzers > 0
	}

	if runPreflights {
		var preflightErr error
		defer func() {
			if preflightErr != nil {
				err := setPreflightResult(appID, sequence, &types.PreflightResults{}, preflightErr)
				if err != nil {
					logger.Error(errors.Wrap(err, "failed to set preflight results"))
					return
				}
			}
		}()

		status, err := store.GetStore().GetDownstreamVersionStatus(appID, sequence)
		if err != nil {
			preflightErr = errors.Wrap(err, "failed to get version status")
			return preflightErr
		}

		if status != storetypes.VersionDeployed && status != storetypes.VersionDeploying {
			if err := store.GetStore().SetDownstreamVersionStatus(appID, sequence, storetypes.VersionPendingPreflight, ""); err != nil {
				preflightErr = errors.Wrapf(err, "failed to set downstream version %d pending preflight", sequence)
				return preflightErr
			}
		}

		collectors, err := registry.UpdateCollectorSpecsWithRegistryData(preflight.Spec.Collectors, registrySettings, renderedKotsKinds.Installation, renderedKotsKinds.License, &renderedKotsKinds.KotsApplication)
		if err != nil {
			preflightErr = errors.Wrap(err, "failed to rewrite images in preflight")
			return preflightErr
		}
		preflight.Spec.Collectors = collectors

		go func() {
			logger.Info("preflight checks beginning")
			uploadPreflightResults, err := execute(appID, sequence, preflight, ignoreRBAC)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to run preflight checks"))
				return
			}

			// Log the preflight results if there are any warnings or errors
			// The app may not get installed so we need to see this info for debugging
			if GetPreflightState(uploadPreflightResults) != "pass" {
				// TODO: Are there conditions when the application gets installed?
				logger.Warnf("Preflight checks completed with warnings or errors. The application may not get installed")
				for _, result := range uploadPreflightResults.Results {
					if result == nil {
						continue
					}
					logger.Infof("preflight state=%s title=%q message=%q", preflightState(*result), result.Title, result.Message)
				}
			} else {
				logger.Info("preflight checks completed")
			}

			go func() {
				err := reporting.GetReporter().SubmitAppInfo(appID) // send app and preflight info when preflights finish
				if err != nil {
					logger.Debugf("failed to submit app info: %v", err)
				}
			}()

			// status could've changed while preflights were running
			status, err := store.GetStore().GetDownstreamVersionStatus(appID, sequence)
			if err != nil {
				logger.Error(errors.Wrapf(err, "failed to check downstream version %d status", sequence))
				return
			}
			if status == storetypes.VersionDeployed || status == storetypes.VersionDeploying || status == storetypes.VersionFailed {
				return
			}

			isDeployed, err := maybeDeployFirstVersion(appID, sequence, uploadPreflightResults)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to deploy first version"))
				return
			}

			// preflight reporting
			if isDeployed {
				if err := reporting.WaitAndReportPreflightChecks(appID, sequence, false, false); err != nil {
					logger.Errorf("failed to send preflights data to replicated app: %v", err)
					return
				}
			}
		}()
	} else if status != storetypes.VersionDeployed && status != storetypes.VersionFailed {
		if sequence == 0 {
			_, err := maybeDeployFirstVersion(appID, sequence, &types.PreflightResults{})
			if err != nil {
				return errors.Wrap(err, "failed to deploy first version")
			}
		} else {
			err := store.GetStore().SetDownstreamVersionStatus(appID, sequence, storetypes.VersionPending, "")
			if err != nil {
				return errors.Wrap(err, "failed to set downstream version status to pending")
			}
		}
	}

	return nil
}

func preflightState(p troubleshootpreflight.UploadPreflightResult) string {
	if p.IsFail {
		return "fail"
	}

	if p.IsWarn {
		return "warn"
	}

	if p.IsPass {
		return "pass"
	}
	return "unknown"
}

// maybeDeployFirstVersion will deploy the first version if preflight checks pass
func maybeDeployFirstVersion(appID string, sequence int64, preflightResults *types.PreflightResults) (bool, error) {
	if sequence != 0 {
		return false, nil
	}

	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get app")
	}

	// do not revert to first version
	if app.CurrentSequence != 0 {
		return false, nil
	}

	preflightState := GetPreflightState(preflightResults)
	if preflightState != "pass" {
		return false, nil
	}

	logger.Debug("automatically deploying first app version")

	err = version.DeployVersion(appID, sequence)
	if err != nil {
		return false, errors.Wrap(err, "failed to deploy version")
	}

	return true, nil
}

func GetPreflightState(preflightResults *types.PreflightResults) string {
	if len(preflightResults.Errors) > 0 {
		return "fail"
	}

	if len(preflightResults.Results) == 0 {
		return "pass"
	}

	state := "pass"
	for _, result := range preflightResults.Results {
		if result.IsFail {
			return "fail"
		} else if result.IsWarn {
			state = "warn"
		}
	}

	return state
}

func GetSpecSecretName(appSlug string) string {
	return fmt.Sprintf("kotsadm-%s-preflight", appSlug)
}

func GetSpecURI(appSlug string) string {
	return fmt.Sprintf("secret/%s/%s", util.PodNamespace, GetSpecSecretName(appSlug))
}

func GetPreflightCommand(appSlug string) []string {
	comamnd := []string{
		"curl https://krew.sh/preflight | bash",
		fmt.Sprintf("kubectl preflight %s\n", GetSpecURI(appSlug)),
	}

	return comamnd
}

func CreateRenderedSpec(app *apptypes.App, sequence int64, origin string, inCluster bool, kotsKinds *kotsutil.KotsKinds, archiveDir string) error {
	var builtPreflight *troubleshootv1beta2.Preflight

	tsKinds, err := kotsutil.LoadTSKindsFromPath(filepath.Join(archiveDir, "rendered"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to load troubleshoot kinds from path: %s", filepath.Join(archiveDir, "rendered")))
	}

	if tsKinds.PreflightsV1Beta2 != nil {
		for _, v := range tsKinds.PreflightsV1Beta2 {
			builtPreflight = troubleshootpreflight.ConcatPreflightSpec(builtPreflight, &v)
		}
	} else {
		builtPreflight = kotsKinds.Preflight.DeepCopy()
	}

	if builtPreflight == nil {
		builtPreflight = &troubleshootv1beta2.Preflight{
			TypeMeta: v1.TypeMeta{
				Kind:       "Preflight",
				APIVersion: "troubleshoot.sh/v1beta2",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "default-preflight",
			},
		}
	}

	builtPreflight.Spec.Collectors = troubleshootcollect.DedupCollectors(builtPreflight.Spec.Collectors)
	builtPreflight.Spec.Analyzers = troubleshootanalyze.DedupAnalyzers(builtPreflight.Spec.Analyzers)

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(app.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get registry settings for app")
	}

	injectDefaultPreflights(builtPreflight, kotsKinds, registrySettings)

	collectors, err := registry.UpdateCollectorSpecsWithRegistryData(builtPreflight.Spec.Collectors, registrySettings, kotsKinds.Installation, kotsKinds.License, &kotsKinds.KotsApplication)
	if err != nil {
		return errors.Wrap(err, "failed to rewrite images in preflight")
	}
	builtPreflight.Spec.Collectors = collectors

	baseURL := os.Getenv("API_ADVERTISE_ENDPOINT")
	if inCluster {
		baseURL = os.Getenv("API_ENDPOINT")
	} else if origin != "" {
		baseURL = origin
	}
	builtPreflight.Spec.UploadResultsTo = fmt.Sprintf("%s/api/v1/preflight/app/%s/sequence/%d", baseURL, app.Slug, sequence)

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(builtPreflight, &b); err != nil {
		return errors.Wrap(err, "failed to encode preflight")
	}

	templatedSpec := b.Bytes()

	renderedSpec, err := helper.RenderAppFile(app, &sequence, templatedSpec, kotsKinds, util.PodNamespace)
	if err != nil {
		return errors.Wrap(err, "failed render preflight spec")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	secretName := GetSpecSecretName(app.Slug)

	existingSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to read preflight secret")
	} else if kuberneteserrors.IsNotFound(err) {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: util.PodNamespace,
				Labels:    kotstypes.GetKotsadmLabels(),
			},
			Data: map[string][]byte{
				SpecDataKey: renderedSpec,
			},
		}

		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create preflight secret")
		}

		return nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}
	existingSecret.Data[SpecDataKey] = renderedSpec
	existingSecret.ObjectMeta.Labels = kotstypes.GetKotsadmLabels()

	_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update preflight secret")
	}

	return nil
}

func injectDefaultPreflights(preflight *troubleshootv1beta2.Preflight, kotskinds *kotsutil.KotsKinds, registrySettings registrytypes.RegistrySettings) {
	if registrySettings.IsValid() && registrySettings.IsReadOnly {
		// Get images from Installation.KnownImages, see UpdateCollectorSpecsWithRegistryData
		images := []string{}
		for _, image := range kotskinds.Installation.Spec.KnownImages {
			images = append(images, image.Image)
		}

		preflight.Spec.Collectors = append(preflight.Spec.Collectors, &troubleshootv1beta2.Collect{
			RegistryImages: &troubleshootv1beta2.RegistryImages{
				Images: images,
			},
		})
		preflight.Spec.Analyzers = append(preflight.Spec.Analyzers, &troubleshootv1beta2.Analyze{
			RegistryImages: &troubleshootv1beta2.RegistryImagesAnalyze{
				AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
					CheckName: "Private Registry Images Available",
				},
				Outcomes: []*troubleshootv1beta2.Outcome{
					{
						Fail: &troubleshootv1beta2.SingleOutcome{
							When:    "missing > 0",
							Message: "Application uses images that cannot be found in the private registry",
						},
					},
					{
						Warn: &troubleshootv1beta2.SingleOutcome{
							When:    "errors > 0",
							Message: "Availability of application images in the private registry could not be verified.",
						},
					},
					{
						Pass: &troubleshootv1beta2.SingleOutcome{
							Message: "All images used by the application are present in the private registry",
						},
					},
				},
			},
		})
	}

	if kotskinds.Installer != nil {
		if deployedInstaller, err := installers.GetDeployedInstaller(); err == nil {
			injectInstallerPreflightIfPresent(preflight, deployedInstaller, kotskinds.Installer)
		} else {
			logger.Error(errors.Wrap(err, "failed to get deployed installer"))
		}
	}

}

func injectInstallerPreflightIfPresent(preflight *troubleshootv1beta2.Preflight, deployedInstaller *kurlv1beta1.Installer, releaseInstaller *kurlv1beta1.Installer) {
	for _, analyzer := range preflight.Spec.Analyzers {
		if analyzer.YamlCompare != nil && analyzer.YamlCompare.Annotations["kots.io/installer"] != "" {
			err := injectInstallerPreflight(preflight, analyzer, deployedInstaller, releaseInstaller)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to inject installer preflight"))
			}
		}
	}
}

func injectInstallerPreflight(preflight *troubleshootv1beta2.Preflight, analyzer *troubleshootv1beta2.Analyze, deployedInstaller *kurlv1beta1.Installer, releaseInstaller *kurlv1beta1.Installer) error {
	if releaseInstaller.Spec.Kurl != nil {
		if releaseInstaller.Spec.Kurl.AdditionalNoProxyAddresses == nil {
			// if this is nil, it will be set to an empty string slice by kurl, so lets do so before comparing
			releaseInstaller.Spec.Kurl.AdditionalNoProxyAddresses = []string{}
		}
	}

	if releaseInstaller.Spec.Kotsadm != nil {
		if releaseInstaller.Spec.Kotsadm.ApplicationSlug == "" {
			// application slug may be injected into the deployed installer, so remove it if not specified in release installer
			deployedInstaller.Spec.Kotsadm.ApplicationSlug = ""
		}
	}

	// Inject deployed installer spec as collected data
	deployedInstallerSpecYaml, err := yaml.Marshal(deployedInstaller.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal deployed installer")
	}

	preflight.Spec.Collectors = append(preflight.Spec.Collectors, &troubleshootv1beta2.Collect{
		Data: &troubleshootv1beta2.Data{
			Name: "kurl/installer.yaml",
			Data: string(deployedInstallerSpecYaml),
		},
	})

	// Inject release installer spec as analyzer value
	releaseInstallerSpecYaml, err := yaml.Marshal(releaseInstaller.Spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal release installer")
	}

	analyzer.YamlCompare.FileName = "kurl/installer.yaml"
	analyzer.YamlCompare.Value = string(releaseInstallerSpecYaml)

	return nil
}
