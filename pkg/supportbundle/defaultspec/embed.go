package defaultspec

import (
	_ "embed"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/supportbundle"
)

//go:embed spec.yaml
var raw []byte

var spec *troubleshootv1beta2.SupportBundle

func init() {
	var err error
	spec, err = supportbundle.ParseSupportBundleFromDoc(raw)
	if err != nil {
		panic(err)
	}
}

func Get() troubleshootv1beta2.SupportBundle {
	return *spec.DeepCopy()
}
