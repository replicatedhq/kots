package base

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/template"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func NewConfigContextTemplateBuilder(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*template.Builder, map[string]template.ItemValue, error) {
	kotsKinds, err := getTemplatingKotsKinds(u)
	if err != nil {
		return nil, nil, err
	}

	var templateContext map[string]template.ItemValue
	if kotsKinds.ConfigValues != nil {
		ctx := map[string]template.ItemValue{}
		for k, v := range kotsKinds.ConfigValues.Spec.Values {
			ctx[k] = template.ItemValue{
				Value:          v.Value,
				Default:        v.Default,
				Filename:       v.Filename,
				RepeatableItem: v.RepeatableItem,
			}
		}
		templateContext = ctx
	} else {
		templateContext = map[string]template.ItemValue{}
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if kotsKinds.Config != nil {
		configGroups = kotsKinds.Config.Spec.Groups
	}

	appInfo := template.ApplicationInfo{
		Slug: renderOptions.AppSlug,
	}

	versionInfo := template.VersionInfo{
		Sequence:                 renderOptions.Sequence,
		Cursor:                   u.UpdateCursor,
		ChannelName:              u.ChannelName,
		VersionLabel:             u.VersionLabel,
		IsRequired:               u.IsRequired,
		ReleaseNotes:             u.ReleaseNotes,
		IsAirgap:                 renderOptions.IsAirgap,
		ReplicatedRegistryDomain: u.ReplicatedRegistryDomain,
		ReplicatedProxyDomain:    u.ReplicatedProxyDomain,
	}

	builderOptions := template.BuilderOptions{
		ConfigGroups:    configGroups,
		ExistingValues:  templateContext,
		LocalRegistry:   renderOptions.RegistrySettings,
		License:         kotsKinds.License,
		Application:     &kotsKinds.KotsApplication,
		VersionInfo:     &versionInfo,
		ApplicationInfo: &appInfo,
		IdentityConfig:  kotsKinds.IdentityConfig,
		Namespace:       renderOptions.Namespace,
		DecryptValues:   true,
	}
	builder, itemValues, err := template.NewBuilder(builderOptions)
	if err != nil {
		return &builder, itemValues, errors.Wrap(err, "failed to create config context")
	}

	return &builder, itemValues, nil
}
