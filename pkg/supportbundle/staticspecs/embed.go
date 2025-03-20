// DO NOT CACHE THE PARSED SPEC
// All getters always parse the bundle from the raw spec (embedded files)
// to ensure that we fetch the latest specs defined in URI fields (if any)

package staticspecs

import (
	_ "embed"

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

func GetVendorSpec(followURI bool) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(vendorspec, followURI)
}

func GetClusterSpecificSpec(followURI bool) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(clusterspec, followURI)
}

func GetDefaultSpec(followURI bool) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(defaultspec, followURI)
}

func GetKurlSpec(followURI bool) (*troubleshootv1beta2.SupportBundle, error) {
	return supportbundle.ParseSupportBundle(kurlspec, followURI)
}
