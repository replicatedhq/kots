package template

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"text/template"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type licenseCtx struct {
	License *kotsv1beta1.License

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
	// return "" for a nil license - it's better than an error, which makes the template engine return "" for the full string
	if ctx.License == nil {
		return ""
	}

	// Update docs at https://github.com/replicatedhq/kots.io/blob/main/content/reference/template-functions/license-context.md
	// when adding new values
	switch name {
	case "isSnapshotSupported":
		return strconv.FormatBool(ctx.License.Spec.IsSnapshotSupported)
	case "IsDisasterRecoverySupported":
		return strconv.FormatBool(ctx.License.Spec.IsDisasterRecoverySupported)
	case "isGitOpsSupported":
		return strconv.FormatBool(ctx.License.Spec.IsGitOpsSupported)
	case "isSupportBundleUploadSupported":
		return strconv.FormatBool(ctx.License.Spec.IsSupportBundleUploadSupported)
	case "isEmbeddedClusterMultinodeDisabled":
		return strconv.FormatBool(ctx.License.Spec.IsEmbeddedClusterMultinodeDisabled)
	case "isIdentityServiceSupported":
		return strconv.FormatBool(ctx.License.Spec.IsIdentityServiceSupported)
	case "isGeoaxisSupported":
		return strconv.FormatBool(ctx.License.Spec.IsGeoaxisSupported)
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
	case "isSemverRequired":
		return strconv.FormatBool(ctx.License.Spec.IsSemverRequired)
	case "customerName":
		return ctx.License.Spec.CustomerName
	case "endpoint":
		return util.ReplicatedAppEndpoint(ctx.License)
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

func (ctx licenseCtx) licenseDockercfg() (string, error) {
	// return "" for a nil license - it's better than an error, which makes the template engine return "" for the full string
	if ctx.License == nil {
		return "", nil
	}

	auth := fmt.Sprintf("%s:%s", ctx.License.Spec.LicenseID, ctx.License.Spec.LicenseID)
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
