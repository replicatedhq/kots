package kotsutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/replicatedhq/troubleshoot/pkg/collect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
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

func TestBundleReferenceCancellation(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		client := fake.NewSimpleClientset()
		client.PrependReactor("get", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("lookup interrupted: %w", cause)
		})
		_, err := prepareBundleReferences(context.Background(), []byte(referenceBundle), client, "app")
		require.ErrorIs(t, err, cause)
		require.Len(t, client.Actions(), 1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := fake.NewSimpleClientset()
	_, err := prepareBundleReferences(ctx, []byte(referenceBundle), client, "app")
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, client.Actions())
}

func TestInvalidBundleBase64DoesNotBlockOtherSpecs(t *testing.T) {
	invalid := "apiVersion: v1\nkind: Secret\nmetadata: {name: invalid-bundle}\ndata:\n  support-bundle-spec: 'not-base64!'\n"
	prepared, err := PrepareSupportBundleReferences([]byte(invalid))
	require.NoError(t, err)
	require.Contains(t, string(prepared), "not-base64!")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bundles.yaml"), []byte(invalid+"\n---\n"+referenceBundle), 0600))
	kinds, err := LoadTSKindsFromPath(dir)
	require.NoError(t, err)
	require.Len(t, kinds.SupportBundlesV1Beta2, 1)
	require.Equal(t, "references", kinds.SupportBundlesV1Beta2[0].Name)
}

func TestPreflightDatabaseFields(t *testing.T) {
	for _, tc := range []struct {
		name, uri, exclude, expected string
		unsupported                  bool
	}{
		{"literal", "{value: 'postgres://database/app'}", "false", "postgres://database/app", false},
		{"secret", "{valueFrom: {secretKeyRef: {name: database, key: uri}}}", "false", "", true},
		{"configmap", "{valueFrom: {configMapKeyRef: {name: database, key: uri}}}", "false", "", true},
		{"excluded", "{valueFrom: {secretKeyRef: {name: missing, key: uri}}}", "true", "", false},
		{"malformed literal", "{value: [invalid]}", "false", "", true},
		{"malformed reference", "{valueFrom: {secretKeyRef: {name: [invalid], key: uri}}}", "false", "", true},
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

func TestMalformedBundleReferenceDoesNotBlockSiblingSpecs(t *testing.T) {
	malformed := strings.Replace(referenceBundle, "secretKeyRef: {name: database, key: uri}", "secretKeyRef: {name: [invalid], key: uri}", 1)
	for _, content := range []string{malformed + "\n---\n" + referenceBundle, referenceBundle + "\n---\n" + malformed} {
		kinds, err := KotsKindsFromMap(map[string][]byte{"bundles.yaml": []byte(content)})
		require.NoError(t, err)
		require.True(t, kinds.HasUnsupportedTroubleshoot)
		require.NotNil(t, kinds.SupportBundle)
		require.Equal(t, "references", kinds.SupportBundle.Name)
		require.Len(t, kinds.SupportBundle.Spec.Collectors, 2)
	}
	// Runtime failures leave other collectors intact and record what was skipped.
	data, err := prepareBundleReferences(context.Background(), []byte(malformed), fake.NewSimpleClientset(), "app")
	require.NoError(t, err)
	resolved, err := LoadSupportBundleFromContents(data)
	require.NoError(t, err)
	require.Equal(t, "retained", resolved.Spec.Collectors[0].Data.Data)
	require.Nil(t, resolved.Spec.Collectors[1].Postgres)
	require.Contains(t, resolved.Spec.Collectors[1].Data.Data, "could not resolve uri")
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
	const credentialSentinel = "postgres://user:must-not-leak-9c2f@database/app"
	secret.Data["uri"] = []byte(credentialSentinel)
	_, err = client.CoreV1().Secrets("app").Update(context.Background(), secret, metav1.UpdateOptions{})
	require.NoError(t, err)
	resolved, err = ResolveSupportBundleReferences(context.Background(), bundle, client, "app")
	require.NoError(t, err)
	require.Equal(t, credentialSentinel, resolved.Spec.Collectors[1].Postgres.URI)
	// Resolve the credential successfully, then fail on the following TLS reference.
	bundle.Annotations[bundleReferencesAnnotation] = strings.Replace(referenceBundle, "\n  analyzers:", "\n    - data: {name: after-database, data: continued}\n  analyzers:", 1)
	require.NoError(t, client.CoreV1().ConfigMaps("shared").Delete(context.Background(), "certificates", metav1.DeleteOptions{}))
	resolved, err = ResolveSupportBundleReferences(context.Background(), bundle, client, "app")
	require.NoError(t, err)
	require.Equal(t, bundle.Spec.Collectors[0], resolved.Spec.Collectors[0])
	require.Nil(t, resolved.Spec.Collectors[1].Postgres)
	require.Contains(t, resolved.Spec.Collectors[1].Data.Data, "could not resolve cacert")
	require.NotContains(t, resolved.Spec.Collectors[1].Data.Data, credentialSentinel)
	require.Equal(t, "continued", resolved.Spec.Collectors[2].Data.Data)
	for _, collector := range resolved.Spec.Collectors {
		output, err := (&collect.CollectData{Collector: collector.Data}).Collect(nil)
		require.NoError(t, err)
		require.Equal(t, collector.Data.Data, string(output[collector.Data.Name]))
		require.NotContains(t, string(output[collector.Data.Name]), credentialSentinel)
	}
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
			require.NoError(t, err)
			if tc.failure != "" {
				require.Nil(t, resolved.Spec.Collectors[0].Postgres)
				require.Contains(t, resolved.Spec.Collectors[0].Data.Data, "could not resolve uri")
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
