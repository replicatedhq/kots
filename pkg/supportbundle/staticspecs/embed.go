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

var spec *troubleshootv1beta2.SupportBundle

func GetVendorSpec() troubleshootv1beta2.SupportBundle {
	var err error
	spec, err = supportbundle.ParseSupportBundleFromDoc(vendorspec)
	if err != nil {
		panic(err)
	}
	return *spec.DeepCopy()
}

func GetClusterSpecificSpec() troubleshootv1beta2.SupportBundle {
	var err error
	spec, err = supportbundle.ParseSupportBundleFromDoc(clusterspec)
	if err != nil {
		panic(err)
	}
	return *spec.DeepCopy()
}

func GetDefaultSpec() troubleshootv1beta2.SupportBundle {
	var err error
	spec, err = supportbundle.ParseSupportBundleFromDoc(defaultspec)
	if err != nil {
		panic(err)
	}
	return *spec.DeepCopy()
}

func GetKurlSpec() troubleshootv1beta2.SupportBundle {
	var err error
	spec, err = supportbundle.ParseSupportBundleFromDoc(kurlspec)
	if err != nil {
		panic(err)
	}
	return *spec.DeepCopy()
}
