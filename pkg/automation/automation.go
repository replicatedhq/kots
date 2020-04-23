package automation

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// AutomateInstall will process any bits left in strategic places
// from the kots install command, so that the admin console
// will finish that installation
func AutomateInstall() error {
	logger.Debug("looking for any automated installs to complete")

	// look for a license secret
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	licenseSecrets, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).List(metav1.ListOptions{
		LabelSelector: "kots.io/automation=license",
	})

	if err != nil {
		return errors.Wrap(err, "failed to list license secrets")
	}

	for _, licenseSecret := range licenseSecrets.Items {
		license, ok := licenseSecret.Data["license"]
		if !ok {
			logger.Errorf("license secret %q does not contain a license field", licenseSecret.Name)
			continue
		}

		kotsLicense, err := kotsutil.LoadLicenseFromBytes(license)
		if err != nil {
			logger.Error(errors.New("license data did not unmarshal"))
			continue
		}

		logger.Debug("automated license install found",
			zap.String("appSlug", kotsLicense.Spec.AppSlug))

		fmt.Printf("%#v\n", kotsLicense)
	}

	return nil
}
