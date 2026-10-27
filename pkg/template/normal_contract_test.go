package template

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

func TestSharedNormalContext(t *testing.T) {
	lic := &licensewrapper.LicenseWrapper{V1: &kotsv1beta1.License{Spec: kotsv1beta1.LicenseSpec{AppSlug: "app", LicenseID: "license", CustomerName: "Customer", IsAirgapSupported: true}}}
	b, _, err := NewBuilder(BuilderOptions{
		License: lic, ApplicationInfo: &ApplicationInfo{Slug: "app"},
		ExistingValues: map[string]ItemValue{"value": {Value: "hello"}, "default": {Default: "fallback"}, "file": {Value: "aGVsbG8=", Filename: "/tmp/file"}},
		VersionInfo:    &VersionInfo{Sequence: 3, Cursor: "7", VersionLabel: "1.2.3", ChannelName: "Stable", ReleaseNotes: "notes", IsRequired: true, ReplicatedProxyDomain: "proxy.replicated.com", ReplicatedRegistryDomain: "registry.replicated.com"},
	})
	require.NoError(t, err)
	cases := map[string]string{
		`ConfigOption "value"`: "hello", `ConfigOption "default"`: "fallback", `ConfigOptionData "file"`: "hello", `ConfigOptionFilename "file"`: "/tmp/file",
		`ConfigOptionEquals "value" "hello"`: "true", `ConfigOptionNotEquals "value" "other"`: "true",
		`LicenseFieldValue "appSlug"`: "app", `LicenseFieldValue "licenseId"`: "license", `LicenseFieldValue "customerName"`: "Customer", `LicenseFieldValue "isAirgapSupported"`: "true",
		`Sequence`: "3", `Cursor`: "7", `VersionLabel`: "1.2.3", `ChannelName`: "Stable", `ReleaseNotes`: "notes", `IsRequired`: "true",
		`ReplicatedProxyDomain`: "proxy.replicated.com", `ReplicatedRegistryDomain`: "registry.replicated.com", `ImagePullSecretName`: "app-registry",
		`ReplicatedImageName "nginx:1" true`: "nginx:1", `ReplicatedImageRegistry "docker.io" true`: "docker.io",
	}
	for expression, want := range cases {
		for _, delimiter := range []string{"repl{{ ", "{{repl "} {
			got, err := b.String(delimiter + expression + " }}")
			require.NoError(t, err)
			require.Equal(t, want, got, expression)
		}
	}
	// Exercise the actual release-level delimiters and pipelines together.
	got, err := b.String("apiVersion: v1\nkind: ConfigMap\ndata:\n  value: 'repl{{ ConfigOption \"value\" | Base64Encode }}'\n  version: '{{repl VersionLabel }}'\n")
	require.NoError(t, err)
	require.Contains(t, got, "value: 'aGVsbG8='")
	require.Contains(t, got, "version: '1.2.3'")
}

func TestSharedGeneratedValues(t *testing.T) {
	b, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	// The random bytes differ; certificate/key pairing, encoding, and cache semantics must agree.
	for _, expression := range []string{
		`TLSCACert "contract-ca" 1`,
		`TLSCert "contract-cert" "localhost" (list "127.0.0.1") (list "localhost") 1`,
		`TLSKey "contract-cert"`,
		`TLSCertFromCA "contract-ca" "contract-signed" "localhost" (list) (list "localhost") 1`,
		`TLSKeyFromCA "contract-ca" "contract-signed" "localhost" (list) (list "localhost") 1`,
	} {
		rendered, err := b.String("repl{{ " + expression + " }}")
		require.NoError(t, err)
		require.Contains(t, rendered, "-----BEGIN")
		again, err := b.String("repl{{ " + expression + " }}")
		require.NoError(t, err)
		require.Equal(t, rendered, again)
	}
	_, err = b.String(`repl{{ KubeSeal "malformed certificate" "app" "name" "value" }}`)
	require.Error(t, err)
}
