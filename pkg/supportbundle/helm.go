package supportbundle

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootclientsetscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/troubleshoot/pkg/docrewrite"
	troubleshootsb "github.com/replicatedhq/troubleshoot/pkg/supportbundle"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSupportBundleSpecFromOCI(licenseID string, url string) (*troubleshootv1beta2.SupportBundle, *troubleshootv1beta2.Redactor, error) {
	err := helm.CreateHelmRegistryCreds(licenseID, licenseID, url)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load collector spec")
	}

	collectorContent, err := troubleshootsb.LoadSupportBundleSpec(url)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load collector spec")
	}

	multidocs := strings.Split(string(collectorContent), "\n---\n")

	// we support both raw collector kinds and supportbundle kinds here
	supportBundle, err := troubleshootsb.ParseSupportBundleFromDoc([]byte(multidocs[0]))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse collector")
	}

	troubleshootclientsetscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode

	redactors := &troubleshootv1beta2.Redactor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Redactor",
			APIVersion: "troubleshoot.sh/v1beta2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-redactor",
		},
	}
	for i, additionalDoc := range multidocs {
		if i == 0 {
			continue
		}
		additionalDoc, err := docrewrite.ConvertToV1Beta2([]byte(additionalDoc))
		if err != nil {
			logger.Infof("failed to convert doc %d to v1beta2: %v", i, err)
			continue
		}
		obj, _, err := decode(additionalDoc, nil, nil)
		if err != nil {
			logger.Infof("failed to parse additional doc %d: %v", i, err)
			continue
		}
		multidocRedactors, ok := obj.(*troubleshootv1beta2.Redactor)
		if !ok {
			continue
		}
		redactors.Spec.Redactors = append(redactors.Spec.Redactors, multidocRedactors.Spec.Redactors...)
	}

	return supportBundle, redactors, nil
}
