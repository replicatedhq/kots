package templates

import (
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

type InstallationContext struct {
	Meta   api.ReleaseMetadata
	Viper  *viper.Viper
	Logger log.Logger
}

func (ctx *InstallationContext) entitlementValue(name string) string {
	if ctx.Meta.Entitlements.Values == nil {
		level.Debug(ctx.Logger).Log("event", "EntitlementValue.empty")
		return ""
	}

	for _, value := range ctx.Meta.Entitlements.Values {
		if value.Key == name {
			return value.Value
		}
	}

	level.Debug(ctx.Logger).Log("event", "EntitlementValue.notFound", "key", name, "values.count", len(ctx.Meta.Entitlements.Values))
	return ""
}

func (ctx *InstallationContext) shipCustomerRelease() string {
	data, err := yaml.Marshal(ctx.Meta)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to marshal release meta", "err", err)
		return ""
	}
	return string(data)
}

func (ctx *InstallationContext) FuncMap() template.FuncMap {
	return template.FuncMap{
		"ShipCustomerRelease": ctx.shipCustomerRelease,
		"EntitlementValue":    ctx.entitlementValue,
		"LicenseFieldValue":   ctx.entitlementValue,
		"Installation": func(name string) string {
			switch name {
			case "state_file_path":
				return constants.StatePath
			case "customer_id":
				return ctx.Meta.CustomerID
			case "semver":
				return ctx.Meta.Semver
			case "channel_name":
				return ctx.Meta.ChannelName
			case "channel_id":
				return ctx.Meta.ChannelID
			case "release_id":
				return ctx.Meta.ReleaseID
			case "installation_id":
				if ctx.Meta.InstallationID != "" {
					// don't warn here, installation_id warnings should happen higher up, closer to the CLI/UX part of the stack
					return ctx.Meta.InstallationID
				}
				level.Warn(ctx.Logger).Log("warning", "template function installation_id is deprecated, please switch to license_id")
				return ctx.Meta.LicenseID
			case "release_notes":
				return ctx.Meta.ReleaseNotes
			case "app_slug":
				return ctx.Meta.AppSlug
			case "license_id":
				if ctx.Meta.LicenseID != "" {
					return ctx.Meta.LicenseID
				}
				level.Warn(ctx.Logger).Log("warning", "license_id not set, falling back to deprecated installation_id")
				return ctx.Meta.InstallationID
			}
			return ""
		},
	}
}
