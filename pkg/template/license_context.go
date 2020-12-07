package template

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"text/template"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

type licenseCtx struct {
	License *kotsv1beta1.License
}

// FuncMap represents the available functions in the licenseCtx.
func (ctx licenseCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"LicenseFieldValue": ctx.licenseFieldValue,
		"LicenseDockerCfg":  ctx.licenseDockercfg,
	}
}

func (ctx licenseCtx) licenseFieldValue(name string) string {
	// return "" for a nil license - it's better than an error, which makes the template engine return "" for the full string
	if ctx.License == nil {
		return ""
	}

	// Update docs at https://github.com/replicatedhq/kots.io/blob/master/content/reference/template-functions/license-context.md
	// when adding new values
	switch name {
	case "isGitOpsSupported":
		return strconv.FormatBool(ctx.License.Spec.IsGitOpsSupported)
	case "isIdentityServiceSupported":
		return strconv.FormatBool(ctx.License.Spec.IsIdentityServiceSupported)
	case "isAirgapSupported":
		return strconv.FormatBool(ctx.License.Spec.IsAirgapSupported)
	case "licenseType":
		return ctx.License.Spec.LicenseType
	case "licenseSequence":
		return strconv.FormatInt(ctx.License.Spec.LicenseSequence, 10)
	case "signature":
		return string(ctx.License.Spec.Signature)
	case "appSlug":
		return ctx.License.Spec.AppSlug
	case "channelID":
		return ctx.License.Spec.ChannelID
	case "channelName":
		return ctx.License.Spec.ChannelName
	case "customerName":
		return ctx.License.Spec.CustomerName
	case "endpoint":
		return ctx.License.Spec.Endpoint
	case "licenseID", "licenseId":
		return ctx.License.Spec.LicenseID
	default:
		entitlement, ok := ctx.License.Spec.Entitlements[name]
		if ok {
			return fmt.Sprintf("%v", entitlement.Value.Value())
		}
		return ""
	}
}

func (ctx licenseCtx) licenseDockercfg() string {
	// return "" for a nil license - it's better than an error, which makes the template engine return "" for the full string
	if ctx.License == nil {
		return ""
	}

	auth := fmt.Sprintf("%s:%s", ctx.License.Spec.LicenseID, ctx.License.Spec.LicenseID)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	dockercfg := map[string]interface{}{
		"auths": map[string]interface{}{
			"proxy.replicated.com": map[string]string{
				"auth": encodedAuth,
			},
			"registry.replicated.com": map[string]string{
				"auth": encodedAuth,
			},
		},
	}

	b, err := json.Marshal(dockercfg)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(b)
}
