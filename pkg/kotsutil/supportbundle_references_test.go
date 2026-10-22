package kotsutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const referenceBundle = `apiVersion: troubleshoot.sh/v1beta3
kind: SupportBundle
metadata:
  name: references
  labels: {vendor: retained}
spec:
  collectors:
    - data: {name: proof, data: retained}
    - postgres:
        collectorName: database
        parameters: [retained]
        uri:
          valueFrom:
            secretKeyRef: {name: database, key: uri}
        tls:
          cacert:
            valueFrom:
              configMapKeyRef: {name: certificates, key: ca, namespace: shared}
          clientCert: {value: certificate}
          clientKey:
            valueFrom:
              secretKeyRef: {name: database, key: key}
  analyzers:
    - clusterVersion: {checkName: retained}
    - textAnalyze: {outcomes: [{fail: {message: 'Unsupported version {{ .version }}'}}]}
  hostCollectors:
    - cpu: {}
  afterCollection:
    - uploadResultsTo: {uri: 'https://example.com/upload'}
`

func TestPreflightDatabaseFields(t *testing.T) {
	for _, tc := range []struct {
		name, uri, exclude, expected string
		unsupported                  bool
	}{
		{"literal", "{value: 'postgres://database/app'}", "false", "postgres://database/app", false},
		{"secret", "{valueFrom: {secretKeyRef: {name: database, key: uri}}}", "false", "", true},
		{"configmap", "{valueFrom: {configMapKeyRef: {name: database, key: uri}}}", "false", "", true},
		{"excluded", "{valueFrom: {secretKeyRef: {name: missing, key: uri}}}", "true", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := []byte("apiVersion: troubleshoot.sh/v1beta3\nkind: Preflight\nspec:\n  collectors:\n    - postgres:\n        uri: " + tc.uri + "\n        exclude: " + tc.exclude + "\n  analyzers:\n    - clusterVersion: {strict: true}\n")
			kinds, err := KotsKindsFromMap(map[string][]byte{"preflight.yaml": doc})
			require.NoError(t, err)
			require.Equal(t, tc.unsupported, kinds.HasUnsupportedTroubleshoot)
			loaded, err := LoadPreflightFromContents(doc)
			require.NoError(t, err)
			require.Equal(t, kinds.Preflight.Spec, loaded.Spec)
			supported := "apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nspec: {uri: 'https://example.com/checks.yaml'}"
			for _, content := range []string{supported + "\n---\n" + string(doc), string(doc) + "\n---\n" + supported} {
				wrapped := "apiVersion: v1\nkind: Secret\nstringData:\n  preflight.yaml: |\n    " + strings.ReplaceAll(strings.TrimSpace(content), "\n", "\n    ") + "\n"
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "preflight-secret.yaml"), []byte(wrapped), 0600))
				discovered, err := LoadTSKindsFromPath(dir)
				require.NoError(t, err)
				merged := &KotsKinds{}
				for i := range discovered.PreflightsV1Beta2 {
					merged.AddPreflight(&discovered.PreflightsV1Beta2[i])
				}
				require.Equal(t, tc.unsupported, merged.HasUnsupportedTroubleshoot)
				if tc.unsupported {
					require.Equal(t, "https://example.com/checks.yaml", merged.Preflight.Spec.Uri)
				}
			}
			if !tc.unsupported {
				require.Equal(t, tc.expected, loaded.Spec.Collectors[0].Postgres.URI)
				require.True(t, loaded.Spec.Analyzers[0].ClusterVersion.Strict.BoolOrDefaultFalse())
			}
		})
	}
}

func TestSupportBundleReferencesAreDeferredAndPreservePolicy(t *testing.T) {
	kinds, err := KotsKindsFromMap(map[string][]byte{"bundle.yaml": []byte(referenceBundle)})
	require.NoError(t, err)
	bundle := kinds.SupportBundle
	require.Empty(t, bundle.Spec.Collectors[1].Postgres.URI)
	require.NotEmpty(t, bundle.Annotations[bundleReferencesAnnotation])
	stored, err := kinds.Marshal("troubleshoot.sh", "v1beta2", "SupportBundle")
	require.NoError(t, err)
	bundle, err = LoadSupportBundleFromContents([]byte(stored))
	require.NoError(t, err)
	client := fake.NewSimpleClientset(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "app"}, Data: map[string][]byte{"uri": []byte("postgres://first"), "key": []byte("private-key")}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "certificates", Namespace: "shared"}, Data: map[string]string{"ca": "ca-certificate"}},
	)
	resolved, err := ResolveSupportBundleReferences(context.Background(), bundle, client, "app")
	require.NoError(t, err)
	expected := bundle.DeepCopy()
	expected.Spec.Collectors[1].Postgres.URI = "postgres://first"
	expected.Spec.Collectors[1].Postgres.TLS.CACert = "ca-certificate"
	expected.Spec.Collectors[1].Postgres.TLS.ClientKey = "private-key"
	expected.Annotations = nil
	require.Equal(t, expected, resolved)
	require.Empty(t, bundle.Spec.Collectors[1].Postgres.URI, "do not persist credentials in the release")
	secret, err := client.CoreV1().Secrets("app").Get(context.Background(), "database", metav1.GetOptions{})
	require.NoError(t, err)
	secret.Data["uri"] = []byte("postgres://changed")
	_, err = client.CoreV1().Secrets("app").Update(context.Background(), secret, metav1.UpdateOptions{})
	require.NoError(t, err)
	resolved, err = ResolveSupportBundleReferences(context.Background(), bundle, client, "app")
	require.NoError(t, err)
	require.Equal(t, "postgres://changed", resolved.Spec.Collectors[1].Postgres.URI)
	_, err = ResolveSupportBundleReferences(context.Background(), bundle, fake.NewSimpleClientset(), "app")
	require.ErrorContains(t, err, "support bundle:")
	require.NotContains(t, err.Error(), "private-key")
}

func TestChartBundleReferencesLoadWithoutClusterAccess(t *testing.T) {
	dir := t.TempDir()
	specs := referenceBundle + "\n---\napiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\nspec:\n  analyzers:\n    - clusterVersion: {}\n"
	content := "apiVersion: v1\nkind: Secret\nstringData:\n  support-bundle-spec: |\n    " + strings.ReplaceAll(strings.TrimSpace(specs), "\n", "\n    ") + "\n"
	file := filepath.Join(dir, "secret.yaml")
	require.NoError(t, os.WriteFile(file, []byte(content), 0600))
	kinds, err := LoadTSKindsFromPath(dir)
	require.NoError(t, err)
	require.Len(t, kinds.SupportBundlesV1Beta2, 1)
	require.Len(t, kinds.PreflightsV1Beta2, 1, "retain other documents in the same Secret")
	require.NotEmpty(t, kinds.SupportBundlesV1Beta2[0].Annotations[bundleReferencesAnnotation])
	require.Equal(t, "retained", kinds.SupportBundlesV1Beta2[0].Spec.Collectors[0].Data.Data)
	unchanged, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, content, string(unchanged))
}

func TestSupportBundleReferenceMissingValues(t *testing.T) {
	for _, tc := range []struct{ name, value, exclude, expected, failure string }{
		{"literal", `{"value":"literal-uri"}`, "false", "literal-uri", ""},
		{"missing secret", `{"valueFrom":{"secretKeyRef":{"name":"absent","key":"uri"}}}`, "false", "", "failed to get secret app/absent"},
		{"missing key", `{"valueFrom":{"secretKeyRef":{"name":"database","key":"uri"}}}`, "false", "", `key "uri" not found`},
		{"optional", `{"valueFrom":{"secretKeyRef":{"name":"absent","key":"uri","optional":true}}}`, "false", "", ""},
		{"excluded bool", `{"valueFrom":{"secretKeyRef":{"name":"absent","key":"uri"}}}`, "true", "", ""},
		{"excluded string", `{"valueFrom":{"secretKeyRef":{"name":"absent","key":"uri"}}}`, `"true"`, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle, err := LoadSupportBundleFromContents([]byte(`{"apiVersion":"troubleshoot.sh/v1beta3","kind":"SupportBundle","metadata":null,"spec":{"collectors":[{"postgres":{"uri":` + tc.value + `,"exclude":` + tc.exclude + `,"tls":{"clientKey":` + tc.value + `}}}]}}`))
			require.NoError(t, err)
			client := fake.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "app"}})
			resolved, err := ResolveSupportBundleReferences(context.Background(), bundle, client, "app")
			if tc.failure != "" {
				require.ErrorContains(t, err, tc.failure)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, resolved.Spec.Collectors[0].Postgres.URI)
			require.Equal(t, tc.expected, resolved.Spec.Collectors[0].Postgres.TLS.ClientKey)
			if strings.HasPrefix(tc.name, "excluded") {
				require.Empty(t, client.Actions(), "excluded collectors must not read the cluster")
			}
		})
	}
}
