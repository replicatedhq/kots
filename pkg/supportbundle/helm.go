package supportbundle

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootclientsetscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/troubleshoot/pkg/docrewrite"
	troubleshootsb "github.com/replicatedhq/troubleshoot/pkg/supportbundle"
	"github.com/segmentio/ksuid"
)

// TODO: support customer redactors
func CollectHelmSupportBundle(appSlug string, licenseID string, url string) (string, error) {
	supportBundleSpec, additionalRedactors, err := getSupportBundleSpecFromOCI(licenseID, url)
	if err != nil {
		return "", errors.Wrapf(err, "failed to download support bundle spec from %s", url)
	}

	randomID, err := ksuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate random bundle id")
	}

	bundleID := strings.ToLower(randomID.String())
	// setSupportBundleProgress(bundleID, supportBundleProgressUpdate{})

	bundle := &types.SupportBundle{
		ID:        bundleID,
		Slug:      bundleID,
		AppID:     appSlug,
		Status:    types.BUNDLE_RUNNING,
		CreatedAt: time.Now(),
		Progress: types.SupportBundleProgress{
			CollectorCount: len(supportBundleSpec.Spec.Collectors),
		},

		BundleSpec:          supportBundleSpec,
		AdditionalRedactors: additionalRedactors,
	}

	err = store.GetStore().CreateInProgressSupportBundle(bundle)
	if err != nil {
		return "", errors.Wrap(err, "failed to ceate support undle in progress")
	}

	progressChan := executeUpdateRoutine(bundle)
	executeSupportBundleCollectRoutine(bundle, progressChan)

	return bundleID, nil
}

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

	additionalRedactors := &troubleshootv1beta2.Redactor{} // TODO: custom redactors
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
		additionalRedactors.Spec.Redactors = append(additionalRedactors.Spec.Redactors, multidocRedactors.Spec.Redactors...)
	}

	return supportBundle, additionalRedactors, nil
}
