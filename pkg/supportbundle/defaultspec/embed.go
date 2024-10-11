package defaultspec

import (
	_ "embed"

	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/supportbundle"
)

//go:embed spec.yaml
var raw []byte

var spec *troubleshootv1beta2.SupportBundle

func Get(isAirgap bool) (troubleshootv1beta2.SupportBundle, error) {
	if spec == nil {
		var err error
		spec, err = supportbundle.ParseSupportBundle(raw, !isAirgap)
		if err != nil {
			return troubleshootv1beta2.SupportBundle{}, err
		}
	}
	return *spec.DeepCopy(), nil
}
