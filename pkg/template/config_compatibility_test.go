package template

import (
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfigDefaultsKeepKOTSIdentityAndNewImageHelpers(t *testing.T) {
	defaults := map[string]string{
		"identity": `repl{{ IdentityServiceClientID }}`,
		"image":    `repl{{ ReplicatedImageName "nginx:1" }}`,
	}
	group := kotsv1beta1.ConfigGroup{Name: "contract"}
	for name, value := range defaults {
		group.Items = append(group.Items, kotsv1beta1.ConfigItem{Name: name, Type: "text", Default: multitype.BoolOrString{Type: multitype.String, StrVal: value}})
	}
	identity := &kotsv1beta1.IdentityConfig{}
	identity.Spec.ClientID = "existing-client"
	_, values, err := NewBuilder(BuilderOptions{ConfigGroups: []kotsv1beta1.ConfigGroup{group}, ExistingValues: map[string]ItemValue{}, ApplicationInfo: &ApplicationInfo{Slug: "app"}, IdentityConfig: identity})
	require.NoError(t, err)
	require.Equal(t, "existing-client", values["identity"].DefaultStr())
	require.Equal(t, "proxy.replicated.com/proxy/app/docker.io/library/nginx:1", values["image"].DefaultStr())
}
