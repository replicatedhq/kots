package identity

import (
	"context"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getWebConfig(ctx context.Context, clientset kubernetes.Interface, namespace string, singleApp bool) (*kotsv1beta1.IdentityWebConfig, error) {
	webConfig := &kotsv1beta1.IdentityWebConfig{
		Title: "Admin Console",
		Theme: &kotsv1beta1.IdentityWebConfigTheme{
			StyleCSSBase64: KotsStyleCSSBase64,
			// LogoURL:        KotsLogoURL,
			LogoBase64:    KotsLogoBase64,
			FaviconBase64: KotsFaviconBase64,
		},
	}

	if !singleApp {
		return webConfig, nil
	}

	brandingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "kotsadm-application-metadata", metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		return webConfig, nil
	} else if err != nil {
		return webConfig, errors.Wrap(err, "failed to get branding config map")
	}

	data, ok := brandingConfigMap.Data["application.yaml"]
	if !ok {
		return webConfig, errors.New("branding config map has no application.yaml")
	}

	// parse data as a kotskind
	application, err := kotsutil.LoadKotsAppFromContents([]byte(data))
	if err != nil {
		return webConfig, errors.Wrap(err, "failed to decode application.yaml")
	}

	if application.Spec.Icon != "" {
		// NOTE: this will not work for base64 icons
		// something to do with the dex templating
		// we will have to override the template
		// <img class="theme-navbar__logo" src="{{ url .ReqPath logo }}">
		webConfig.Theme.LogoURL = application.Spec.Icon
	}
	if application.Spec.Title != "" {
		webConfig.Title = application.Spec.Title
	}
	// TODO: we don't really support base64 here for favicon

	return webConfig, nil
}
