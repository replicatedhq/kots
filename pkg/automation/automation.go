package automation

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
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
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// AutomateInstall will process any bits left in strategic places
// from the kots install command, so that the admin console
// will finish that installation
func AutomateInstall() error {
	logger.Debug("looking for any automated installs to complete")

	// look for a license secret
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	licenseSecrets, err := clientset.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "kots.io/automation=license",
	})

	if err != nil {
		return errors.Wrap(err, "failed to list license secrets")
	}

	cleanup := func(licenseSecret *corev1.Secret, appSlug string) {
		err = kotsutil.RemoveAppVersionLabelFromInstallationParams(kotsadmtypes.KotsadmConfigMap)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to delete app version label from config"))
		}

		err = clientset.CoreV1().Secrets(licenseSecret.Namespace).Delete(context.TODO(), licenseSecret.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to delete license data %s", licenseSecret.Name))
			// this is going to create a new app on each start now!
		}

		err = deleteAirgapData(clientset, appSlug)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to delete airgap data"))
		}
	}

LICENSE_LOOP:
	for _, licenseSecret := range licenseSecrets.Items {

		license, ok := licenseSecret.Data["license"]
		if !ok {
			logger.Errorf("license secret %q does not contain a license field", licenseSecret.Name)
			continue
		}

		unverifiedLicense, err := kotsutil.LoadLicenseFromBytes(license)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to unmarshal license data"))
			appSlug := ""
			if licenseSecret.Labels != nil {
				appSlug = licenseSecret.Labels["kots.io/app"]
			}
			cleanup(&licenseSecret, appSlug)
			continue
		}

		logger.Debug("automated license install found",
			zap.String("appSlug", unverifiedLicense.Spec.AppSlug))

		verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to verify license signature"))
			cleanup(&licenseSecret, unverifiedLicense.Spec.AppSlug)
			continue
		}

		if !kotsadm.IsAirgap() {
			licenseData, err := kotslicense.GetLatestLicense(verifiedLicense)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to get latest license"))
				continue
			}
			verifiedLicense = licenseData.License
			license = licenseData.LicenseBytes
		}

		// check license expiration
		expired, err := kotspull.LicenseIsExpired(verifiedLicense)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to check if license is expired for app %s", verifiedLicense.Spec.AppSlug))
			cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
			continue
		}
		if expired {
			logger.Errorf("license is expired for app %s", verifiedLicense.Spec.AppSlug)
			cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
			continue
		}

		// check if license already exists
		existingLicense, err := kotsadmlicense.CheckIfLicenseExists(license)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to check if license already exists for app %s", verifiedLicense.Spec.AppSlug))
			cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
			continue
		}
		if existingLicense != nil {
			resolved, err := kotslicense.ResolveExistingLicense(verifiedLicense)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to resolve existing license conflict"))
			}
			if !resolved {
				logger.Errorf("license already exists for app %s", verifiedLicense.Spec.AppSlug)
				cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
				continue
			}
		}

		instParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to get existing kotsadm config map"))
			continue
		}

		desiredAppName := strings.Replace(verifiedLicense.Spec.AppSlug, "-", " ", 0)
		upstreamURI := fmt.Sprintf("replicated://%s", verifiedLicense.Spec.AppSlug)

		a, err := store.GetStore().CreateApp(desiredAppName, upstreamURI, string(license), verifiedLicense.Spec.IsAirgapSupported, instParams.SkipImagePush, instParams.RegistryIsReadOnly)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to create app record"))
			continue
		}

		// airgap data is the airgap manifest + app specs + image list laoded from configmaps
		airgapData, err := getAirgapData(clientset, verifiedLicense)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to load airgap data for %s", verifiedLicense.Spec.AppSlug))
			continue
		}

		// check for the airgap flag in the annotations
		objMeta := licenseSecret.GetObjectMeta()
		annotations := objMeta.GetAnnotations()
		if instParams.SkipImagePush && airgapData != nil {
			// Images have been pushed and there is airgap app data available, so this is an airgap install.
			airgapFilesDir, err := ioutil.TempDir("", "headless-airgap")
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to create temp dir"))
				continue
			}
			defer os.RemoveAll(airgapFilesDir)

			for filename, data := range airgapData {
				err := ioutil.WriteFile(filepath.Join(airgapFilesDir, filename), data, 0644)
				if err != nil {
					logger.Error(errors.Wrapf(err, "failed to create file %s", filename))
					continue LICENSE_LOOP
				}
			}

			kotsadmOpts, err := kotsadm.GetKotsadmOptionsFromCluster(util.PodNamespace, clientset)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to load registry info"))
				continue
			}

			createAppOpts := airgap.CreateAirgapAppOpts{
				PendingApp: &airgaptypes.PendingApp{
					ID:          a.ID,
					Slug:        a.Slug,
					Name:        a.Name,
					LicenseData: string(license),
				},
				AirgapPath:             airgapFilesDir,
				RegistryHost:           kotsadmOpts.OverrideRegistry,
				RegistryNamespace:      kotsadmOpts.OverrideNamespace,
				RegistryUsername:       kotsadmOpts.Username,
				RegistryPassword:       kotsadmOpts.Password,
				RegistryIsReadOnly:     instParams.RegistryIsReadOnly,
				IsAutomated:            true,
				SkipPreflights:         instParams.SkipPreflights,
				SkipCompatibilityCheck: instParams.SkipCompatibilityCheck,
			}
			err = airgap.CreateAppFromAirgap(createAppOpts)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to create airgap app"))
				continue
			}
		} else if annotations["kots.io/airgap"] != "true" {
			// Otherwise there is no airgap data, so this is an online install.
			createAppOpts := online.CreateOnlineAppOpts{
				PendingApp: &onlinetypes.PendingApp{
					ID:           a.ID,
					Slug:         a.Slug,
					Name:         a.Name,
					LicenseData:  string(license),
					VersionLabel: instParams.AppVersionLabel,
				},
				UpstreamURI:            upstreamURI,
				IsAutomated:            true,
				SkipPreflights:         instParams.SkipPreflights,
				SkipCompatibilityCheck: instParams.SkipCompatibilityCheck,
			}
			_, err := online.CreateAppFromOnline(createAppOpts)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to create online app"))
				continue
			}
		}

		cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
	}

	return nil
}

func AirgapInstall(appSlug string, additionalFiles map[string][]byte) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}

	cleanup := func(licenseSecret *corev1.Secret, appSlug string) {
		err = clientset.CoreV1().Secrets(licenseSecret.Namespace).Delete(context.TODO(), licenseSecret.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to delete license data %s", licenseSecret.Name))
		}

		err = deleteAirgapData(clientset, appSlug)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to delete airgap data"))
		}
	}

	selectorLabels := map[string]string{
		"kots.io/automation": "license",
		"kots.io/app":        appSlug,
	}
	licenseSecrets, err := clientset.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list license secrets")
	}

	if len(licenseSecrets.Items) != 1 {
		return errors.Errorf("expected one license for app %s, but found %d", appSlug, len(licenseSecrets.Items))
	}

	licenseSecret := licenseSecrets.Items[0]
	license, ok := licenseSecret.Data["license"]
	if !ok {
		appSlug := ""
		if licenseSecret.Labels != nil {
			appSlug = licenseSecret.Labels["kots.io/app"]
		}
		cleanup(&licenseSecret, appSlug)
		return errors.Errorf("license secret %q does not contain a license field", licenseSecret.Name)
	}

	unverifiedLicense, err := kotsutil.LoadLicenseFromBytes(license)
	if err != nil {
		cleanup(&licenseSecret, unverifiedLicense.Spec.AppSlug)
		return errors.Wrap(err, "failed to unmarshal license data")
	}

	logger.Debug("automated license install found",
		zap.String("appSlug", unverifiedLicense.Spec.AppSlug))

	verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
	if err != nil {
		cleanup(&licenseSecret, unverifiedLicense.Spec.AppSlug)
		return errors.Wrap(err, "failed to verify license signature")
	}

	// check license expiration
	expired, err := kotspull.LicenseIsExpired(verifiedLicense)
	if err != nil {
		cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
		return errors.Wrapf(err, "failed to check is license is expired for app %s", verifiedLicense.Spec.AppSlug)
	}
	if expired {
		cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
		return errors.Errorf("license is expired for app %s", verifiedLicense.Spec.AppSlug)
	}

	// check if license already exists
	existingLicense, err := kotsadmlicense.CheckIfLicenseExists(license)
	if err != nil {
		cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
		return errors.Wrapf(err, "failed to check if license already exists for app %s", verifiedLicense.Spec.AppSlug)
	}
	if existingLicense != nil {
		resolved, err := kotslicense.ResolveExistingLicense(verifiedLicense)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to resolve existing license conflict"))
		}
		if !resolved {
			cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)
			return errors.Errorf("license already exists for app %s", verifiedLicense.Spec.AppSlug)
		}
	}

	instParams, err := kotsutil.GetInstallationParams(kotsadmtypes.KotsadmConfigMap)
	if err != nil {
		return errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	desiredAppName := strings.Replace(verifiedLicense.Spec.AppSlug, "-", " ", 0)
	upstreamURI := fmt.Sprintf("replicated://%s", verifiedLicense.Spec.AppSlug)

	a, err := store.GetStore().CreateApp(desiredAppName, upstreamURI, string(license), verifiedLicense.Spec.IsAirgapSupported, instParams.SkipImagePush, instParams.RegistryIsReadOnly)
	if err != nil {
		return errors.Wrap(err, "failed to create app record")
	}

	// airgap data is the airgap manifest + app specs + image list laoded from configmaps
	airgapData, err := getAirgapData(clientset, verifiedLicense)
	if err != nil {
		return errors.Wrapf(err, "failed to load airgap data for %s", verifiedLicense.Spec.AppSlug)
	}

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

	kotsadmOpts, err := kotsadm.GetKotsadmOptionsFromCluster(util.PodNamespace, clientset)
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
		AirgapPath:             airgapFilesDir,
		RegistryHost:           kotsadmOpts.OverrideRegistry,
		RegistryNamespace:      kotsadmOpts.OverrideNamespace,
		RegistryUsername:       kotsadmOpts.Username,
		RegistryPassword:       kotsadmOpts.Password,
		RegistryIsReadOnly:     instParams.RegistryIsReadOnly,
		IsAutomated:            true,
		SkipPreflights:         instParams.SkipPreflights,
		SkipCompatibilityCheck: instParams.SkipCompatibilityCheck,
	}
	err = airgap.CreateAppFromAirgap(createAppOpts)
	if err != nil {
		return errors.Wrap(err, "failed to create airgap app")
	}

	cleanup(&licenseSecret, verifiedLicense.Spec.AppSlug)

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
