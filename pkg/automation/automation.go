package automation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/airgap"
	airgaptypes "github.com/replicatedhq/kots/pkg/airgap/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmlicense "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/online"
	onlinetypes "github.com/replicatedhq/kots/pkg/online/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/tasks"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	AutomatedInstallRunning = "running"
	AutomatedInstallSuccess = "success"
	AutomatedInstallFailed  = "failed"
)

type AutomateInstallOptions struct {
	AppSlug         string
	AdditionalFiles map[string][]byte
}

type AutomateInstallTaskMessage struct {
	Message       string                             `json:"message"`
	VersionStatus storetypes.DownstreamVersionStatus `json:"versionStatus"`
	Error         string                             `json:"error"`
}

// AutomateInstall will process any bits left in strategic places
// from the kots install command, so that the admin console
// will finish that installation
func AutomateInstall(opts AutomateInstallOptions) error {
	logger.Debug("looking for any automated installs to complete")

	// look for a license secret
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	selectorLabels := map[string]string{
		"kots.io/automation": "license",
	}
	if opts.AppSlug != "" {
		selectorLabels["kots.io/app"] = opts.AppSlug
	}
	licenseSecrets, err := clientset.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list license secrets")
	}

	if opts.AppSlug != "" && len(licenseSecrets.Items) != 1 {
		return errors.Errorf("expected one license for app %s, but found %d", opts.AppSlug, len(licenseSecrets.Items))
	}

	cleanup := func(licenseSecret *corev1.Secret) {
		err = kotsutil.RemoveAppVersionLabelFromInstallationParams(kotsadmtypes.KotsadmConfigMap)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to delete app version label from config"))
		}

		err = clientset.CoreV1().Secrets(licenseSecret.Namespace).Delete(context.TODO(), licenseSecret.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to delete license data %s", licenseSecret.Name))
			// this is going to create a new app on each start now!
		}

		license, ok := licenseSecret.Data["license"]
		if !ok {
			logger.Error(fmt.Errorf("license secret %q does not contain a license field", licenseSecret.Name))
		}

		decodedLicense, err := kotsutil.LoadLicenseFromBytes(license)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to unmarshal license data"))
		}

		if decodedLicense != nil {
			err = deleteAirgapData(clientset, decodedLicense.Spec.AppSlug)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to delete airgap data"))
			}
		}
	}

	for _, licenseSecret := range licenseSecrets.Items {
		err := installLicenseSecret(clientset, licenseSecret, opts.AdditionalFiles)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to install license %s", licenseSecret.Name))
		}
		cleanup(&licenseSecret)
	}

	return nil
}

func installLicenseSecret(clientset *kubernetes.Clientset, licenseSecret corev1.Secret, additionalFiles map[string][]byte) (finalError error) {
	license, ok := licenseSecret.Data["license"]
	if !ok {
		return fmt.Errorf("license secret %q does not contain a license field", licenseSecret.Name)
	}

	unverifiedLicense, err := kotsutil.LoadLicenseFromBytes(license)
	appSlug := ""
	if err != nil {
		if licenseSecret.Labels != nil {
			appSlug = licenseSecret.Labels["kots.io/app"]
		}
		return errors.Wrap(err, "failed to unmarshal license data")
	}
	appSlug = unverifiedLicense.Spec.AppSlug

	logger.Debug("automated license install found",
		zap.String("appSlug", appSlug))

	taskMessage, err := json.Marshal(AutomateInstallTaskMessage{Message: "Installing app..."})
	if err != nil {
		return errors.Wrap(err, "failed to marshal task message")
	}
	taskID := fmt.Sprintf("automated-install-slug-%s", appSlug)
	if err := tasks.SetTaskStatus(taskID, string(taskMessage), AutomatedInstallRunning); err != nil {
		logger.Error(errors.Wrap(err, "failed to set task status"))
	}

	finishedCh := make(chan struct{})
	defer close(finishedCh)
	go func() {
		for {
			select {
			case <-time.After(time.Second * 2):
				if err := tasks.UpdateTaskStatusTimestamp(taskID); err != nil {
					logger.Error(errors.Wrapf(err, "failed to update task %s", taskID))
				}
			case <-finishedCh:
				return
			}
		}
	}()

	defer func() {
		if finalError == nil {
			appID, err := store.GetStore().GetAppIDFromSlug(appSlug)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to get app id from slug"))
			}
			status, err := store.GetStore().GetDownstreamVersionStatus(appID, 0)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to get downstream version status"))
			}
			taskMessage, err := json.Marshal(AutomateInstallTaskMessage{
				Message:       "Install complete",
				VersionStatus: status,
			})
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to marshal task message"))
			}
			if err := tasks.SetTaskStatus(taskID, string(taskMessage), AutomatedInstallSuccess); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
		} else {
			taskMessage, err := json.Marshal(AutomateInstallTaskMessage{Error: finalError.Error()})
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to marshal task message"))
			}
			if err := tasks.SetTaskStatus(taskID, string(taskMessage), AutomatedInstallFailed); err != nil {
				logger.Error(errors.Wrap(err, "failed to set error on install task status"))
			}
		}
	}()

	verifiedLicense, err := kotslicense.VerifySignature(unverifiedLicense)
	if err != nil {
		return errors.Wrap(err, "failed to verify license signature")
	}

	if !kotsadm.IsAirgap() {
		licenseData, err := replicatedapp.GetLatestLicense(verifiedLicense)
		if err != nil {
			return errors.Wrap(err, "failed to get latest license")
		}
		verifiedLicense = licenseData.License
		license = licenseData.LicenseBytes
	}

	// check license expiration
	expired, err := kotslicense.LicenseIsExpired(verifiedLicense)
	if err != nil {
		return errors.Wrapf(err, "failed to check if license is expired for app %s", appSlug)
	}
	if expired {
		return fmt.Errorf("license is expired for app %s", appSlug)
	}

	// check if license already exists
	existingLicense, err := kotsadmlicense.CheckIfLicenseExists(license)
	if err != nil {
		return errors.Wrapf(err, "failed to check if license already exists for app %s", appSlug)
	}
	if existingLicense != nil {
		resolved, err := kotsadmlicense.ResolveExistingLicense(verifiedLicense)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to resolve existing license conflict"))
		}
		if !resolved {
			return fmt.Errorf("license already exists for app %s", appSlug)
		}
	}

	instParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		return errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	desiredAppName := strings.Replace(appSlug, "-", " ", 0)
	upstreamURI := fmt.Sprintf("replicated://%s", appSlug)

	matchedChannelID, err := kotsutil.FindChannelIDInLicense(instParams.RequestedChannelSlug, verifiedLicense)
	if err != nil {
		return errors.Wrap(err, "failed to find requested channel in license")
	}

	a, err := store.GetStore().CreateApp(desiredAppName, matchedChannelID, upstreamURI, string(license), verifiedLicense.Spec.IsAirgapSupported, instParams.SkipImagePush, instParams.RegistryIsReadOnly)
	if err != nil {
		return errors.Wrap(err, "failed to create app record")
	}
	appSlug = a.Slug

	// airgap data is the airgap manifest + app specs + image list loaded from configmaps
	airgapData, err := getAirgapData(clientset, verifiedLicense)
	if err != nil {
		return errors.Wrapf(err, "failed to load airgap data for %s", appSlug)
	}

	// check for the airgap flag in the annotations
	objMeta := licenseSecret.GetObjectMeta()
	annotations := objMeta.GetAnnotations()
	if instParams.SkipImagePush && airgapData != nil {
		if len(airgapData) == 0 {
			return errors.Errorf("failed to find airgap automation data")
		}

		for k, v := range additionalFiles {
			airgapData[k] = v
		}

		// Images have been pushed and there is airgap app data available, so this is an airgap install.
		airgapFilesDir, err := ioutil.TempDir("", "headless-airgap")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(airgapFilesDir)

		for filename, data := range airgapData {
			err := ioutil.WriteFile(filepath.Join(airgapFilesDir, filename), data, 0644)
			if err != nil {
				return errors.Wrapf(err, "failed to create file %s", filename)
			}
		}

		registryConfig, err := kotsadm.GetRegistryConfigFromCluster(util.PodNamespace, clientset)
		if err != nil {
			return errors.Wrap(err, "failed to load registry info")
		}

		createAppOpts := airgap.CreateAirgapAppOpts{
			PendingApp: &airgaptypes.PendingApp{
				ID:          a.ID,
				Slug:        a.Slug,
				Name:        a.Name,
				LicenseData: string(license),
			},
			AirgapRootDir:          airgapFilesDir,
			RegistryHost:           registryConfig.OverrideRegistry,
			RegistryNamespace:      registryConfig.OverrideNamespace,
			RegistryUsername:       registryConfig.Username,
			RegistryPassword:       registryConfig.Password,
			RegistryIsReadOnly:     instParams.RegistryIsReadOnly,
			IsAutomated:            true,
			SkipPreflights:         instParams.SkipPreflights,
			SkipCompatibilityCheck: instParams.SkipCompatibilityCheck,
		}
		err = airgap.CreateAppFromAirgap(createAppOpts)
		if err != nil {
			return errors.Wrap(err, "failed to create airgap app")
		}
	} else if annotations["kots.io/airgap"] != "true" {
		createAppOpts := online.CreateOnlineAppOpts{
			PendingApp: &onlinetypes.PendingApp{
				ID:           a.ID,
				Slug:         a.Slug,
				Name:         a.Name,
				LicenseData:  string(license),
				VersionLabel: instParams.AppVersionLabel,
				ChannelID:    a.SelectedChannelID,
			},
			UpstreamURI:            upstreamURI,
			IsAutomated:            true,
			SkipPreflights:         instParams.SkipPreflights,
			SkipCompatibilityCheck: instParams.SkipCompatibilityCheck,
		}
		_, err = online.CreateAppFromOnline(createAppOpts)
		if err != nil {
			return errors.Wrap(err, "failed to create online app")
		}
	}

	return nil
}

func NeedToWaitForAirgapApp() (bool, error) {
	logger.Debug("looking for any automated installs to complete")

	// look for a license secret
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get k8s client set")
	}

	licenseSecrets, err := clientset.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "kots.io/automation=license",
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list license secrets")
	}

	for _, licenseSecret := range licenseSecrets.Items {
		license, ok := licenseSecret.Data["license"]
		if !ok {
			logger.Errorf("license secret %q does not contain a license field", licenseSecret.Name)
			continue
		}

		unverifiedLicense, err := kotsutil.LoadLicenseFromBytes(license)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to unmarshal license data"))
			continue
		}

		// airgap data is the airgap manifest + app specs + image list laoded from configmaps
		needToWait, err := needToWaitForAirgapApp(clientset, unverifiedLicense)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to load airgap data for %s", unverifiedLicense.Spec.AppSlug))
			continue
		}

		if needToWait {
			return true, nil
		}
	}

	return false, nil
}

func getAirgapData(clientset kubernetes.Interface, license *kotsv1beta1.License) (map[string][]byte, error) {
	selectorLabels := map[string]string{
		"kots.io/automation": "airgap",
		"kots.io/app":        license.Spec.AppSlug,
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list configmaps")
	}

	result := map[string][]byte{}
	for _, configMap := range configMaps.Items {
		for key, value := range configMap.Data {
			if key == "wait-for-airgap-app" { // do this better
				continue
			}
			decoded, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode configmap value")
			}
			result[key] = decoded
		}
	}

	return result, nil
}

func needToWaitForAirgapApp(clientset kubernetes.Interface, license *kotsv1beta1.License) (bool, error) {
	selectorLabels := map[string]string{
		"kots.io/automation": "airgap",
		"kots.io/app":        license.Spec.AppSlug,
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list configmaps")
	}

	for _, configMap := range configMaps.Items {
		value, ok := configMap.Data["wait-for-airgap-app"]
		if !ok {
			continue
		}

		b, _ := strconv.ParseBool(value)
		return b, nil
	}

	return false, nil
}

func deleteAirgapData(clientset kubernetes.Interface, appSlug string) error {
	if appSlug == "" {
		return nil
	}

	selectorLabels := map[string]string{
		"kots.io/automation": "airgap",
		"kots.io/app":        appSlug,
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list configmaps")
	}

	for _, configMap := range configMaps.Items {
		err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Delete(context.TODO(), configMap.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete configmap %s", configMap.Name)
		}
	}

	return nil
}
