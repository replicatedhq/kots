// DO NOT CACHE THE PARSED SPEC
// All getters always parse the bundle from the raw spec (embedded files)
// to ensure that we fetch the latest specs defined in URI fields (if any)

package staticspecs

import (
	_ "embed"

	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/supportbundle"
)

//go:embed clusterspec.yaml
var clusterspec []byte

//go:embed kurlspec.yaml
var kurlspec []byte

//go:embed vendorspec.yaml
var vendorspec []byte

//go:embed defaultspec.yaml
var defaultspec []byte

func GetVendorSpec(isAirgap bool) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(vendorspec, !isAirgap)
}

func GetClusterSpecificSpec(app *apptypes.App) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(clusterspec, !app.IsAirgap)
}

func GetDefaultSpec(app *apptypes.App) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(defaultspec, !app.IsAirgap)
}

func GetKurlSpec(app *apptypes.App) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(kurlspec, !app.IsAirgap)
}
