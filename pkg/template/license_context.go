package template

import (
	"fmt"
	"text/template"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

type LicenseCtx struct {
	License *kotsv1beta1.License
}

// FuncMap represents the available functions in the LicenseCtx.
func (ctx LicenseCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"LicenseFieldValue": ctx.licenseFieldValue,
	}
}

func (ctx LicenseCtx) licenseFieldValue(name string) string {
	for key, entitlement := range ctx.License.Spec.Entitlements {
		if key == name {
			return fmt.Sprintf("%v", entitlement.Value.Value())
		}
	}
	return ""
}
