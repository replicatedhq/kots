package template

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"text/template"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

type licenseCtx struct {
	License *licensewrapper.LicenseWrapper

	// DEPRECATED: App is optional, but if provided, it will be used to determine the registry domains
	App *kotsv1beta1.Application

	// VersionInfo is optional, but if provided, it will be used to determine the registry domains
	VersionInfo *VersionInfo
}

// FuncMap represents the available functions in the licenseCtx.
func (ctx licenseCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"LicenseFieldValue": ctx.licenseFieldValue,
		"LicenseDockerCfg":  ctx.licenseDockercfg,
	}
}

func (ctx licenseCtx) licenseFieldValue(name string) string {
	// return "" for a nil/empty license - it's better than an error, which makes the template engine return "" for the full string
	if ctx.License.IsEmpty() {
		return ""
	}

	// Update docs at https://github.com/replicatedhq/kots.io/blob/main/content/reference/template-functions/license-context.md
	// when adding new values
	switch name {
	case "isSnapshotSupported":
		return strconv.FormatBool(ctx.License.IsSnapshotSupported())
	case "IsDisasterRecoverySupported":
		return strconv.FormatBool(ctx.License.IsDisasterRecoverySupported())
	case "isGitOpsSupported":
		return strconv.FormatBool(ctx.License.IsGitOpsSupported())
	case "isSupportBundleUploadSupported":
		return strconv.FormatBool(ctx.License.IsSupportBundleUploadSupported())
	case "isEmbeddedClusterMultiNodeEnabled":
		return strconv.FormatBool(ctx.License.IsEmbeddedClusterMultiNodeEnabled())
	case "isIdentityServiceSupported":
		return strconv.FormatBool(ctx.License.IsIdentityServiceSupported())
	case "isGeoaxisSupported":
		return strconv.FormatBool(ctx.License.IsGeoaxisSupported())
	case "isAirgapSupported":
		return strconv.FormatBool(ctx.License.IsAirgapSupported())
	case "licenseType":
		return ctx.License.GetLicenseType()
	case "licenseSequence":
		return strconv.FormatInt(ctx.License.GetLicenseSequence(), 10)
	case "signature":
		return string(ctx.License.GetSignature())
	case "appSlug":
		return ctx.License.GetAppSlug()
	case "channelID":
		return ctx.License.GetChannelID()
	case "channelName":
		return ctx.License.GetChannelName()
	case "isSemverRequired":
		return strconv.FormatBool(ctx.License.IsSemverRequired())
	case "customerName":
		return ctx.License.GetCustomerName()
	case "endpoint":
		endpoint := ctx.License.GetEndpoint()
		if endpoint == "" {
			endpoint = "https://replicated.app"
		}
		return endpoint
	case "licenseID", "licenseId":
		return ctx.License.GetLicenseID()
	default:
		entitlements := ctx.License.GetEntitlements()
		entitlement, ok := entitlements[name]
		if ok {
			value := entitlement.GetValue()
			return fmt.Sprintf("%v", (&value).Value())
		}
		return ""
	}
}

func (ctx licenseCtx) licenseDockercfg() (string, error) {
	// return "" for a nil/empty license - it's better than an error, which makes the template engine return "" for the full string
	if ctx.License.IsEmpty() {
		return "", nil
	}

	licenseID := ctx.License.GetLicenseID()
	auth := fmt.Sprintf("%s:%s", "LICENSE_ID", licenseID)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	installation := &kotsv1beta1.Installation{
		Spec: InstallationSpecFromVersionInfo(ctx.VersionInfo),
	}
	registryProxyInfo, err := registry.GetRegistryProxyInfo(ctx.License, installation, ctx.App)
	if err != nil {
		return "", errors.Wrap(err, "get registry proxy info")
	}

	dockercfg := map[string]interface{}{
		"auths": map[string]interface{}{
			registryProxyInfo.Proxy: map[string]string{
				"auth": encodedAuth,
			},
			registryProxyInfo.Registry: map[string]string{
				"auth": encodedAuth,
			},
		},
	}

	b, err := json.Marshal(dockercfg)
	if err != nil {
		// TODO: log
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
