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
	// GENERAL FIELDS
	case "appSlug":
		return ctx.License.GetAppSlug()
	case "channelID":
		return ctx.License.GetChannelID()
	case "channelName":
		return ctx.License.GetChannelName()
	case "customerEmail":
		return ctx.License.GetCustomerEmail()
	case "endpoint":
		endpoint := ctx.License.GetEndpoint()
		if endpoint == "" {
			endpoint = "https://replicated.app"
		}
		return endpoint
	case "licenseID", "licenseId":
		return ctx.License.GetLicenseID()
	case "licenseSequence":
		return strconv.FormatInt(ctx.License.GetLicenseSequence(), 10)
	case "customerID":
		return ctx.License.GetCustomerID()
	case "customerName":
		return ctx.License.GetCustomerName()
	case "signature":
		return string(ctx.License.GetSignature())
	case "licenseType":
		return ctx.License.GetLicenseType()
	case "replicatedProxyDomain":
		return ctx.License.GetReplicatedProxyDomain()
	// INSTALL TYPES
	case "isEmbeddedClusterDownloadEnabled":
		return strconv.FormatBool(ctx.License.IsEmbeddedClusterDownloadEnabled())
	// INSTALL OPTIONS
	case "isAirgapSupported":
		return strconv.FormatBool(ctx.License.IsAirgapSupported())
	case "isEmbeddedClusterMultiNodeEnabled":
		return strconv.FormatBool(ctx.License.IsEmbeddedClusterMultiNodeEnabled())
	// ADMIN CONSOLE FEATURE OPTIONS
	// there was a bug where this one started ithe a capital I but I don't
	// want to remove it in case some vendors are using it that way
	case "IsDisasterRecoverySupported", "isDisasterRecoverySupported":
		return strconv.FormatBool(ctx.License.IsDisasterRecoverySupported())
	case "isGeoaxisSupported":
		return strconv.FormatBool(ctx.License.IsGeoaxisSupported())
	case "isGitOpsSupported":
		return strconv.FormatBool(ctx.License.IsGitOpsSupported())
	case "isIdentityServiceSupported":
		return strconv.FormatBool(ctx.License.IsIdentityServiceSupported())
	case "isSemverRequired":
		return strconv.FormatBool(ctx.License.IsSemverRequired())
	case "isSnapshotSupported":
		return strconv.FormatBool(ctx.License.IsSnapshotSupported())
	case "isSupportBundleUploadSupported":
		return strconv.FormatBool(ctx.License.IsSupportBundleUploadSupported())
		// ENTITLEMENT FIELDS (a.k.a custom license fields)
	default:
		entitlements := ctx.License.GetEntitlements()
		entitlement, ok := entitlements[name]
		if ok {
			value := entitlement.GetValue()
			return fmt.Sprintf("%v", value)
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
