package base

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"

	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// decodeHelmReleaseSecret decodes a Helm release secret YAML and returns the
// parsed Secret object plus the decoded release JSON bytes.
func decodeHelmReleaseSecret(t *testing.T, secretYAML []byte) (*v1.Secret, map[string]interface{}) {
	t.Helper()
	var secret v1.Secret
	if err := yaml.Unmarshal(secretYAML, &secret); err != nil {
		t.Fatalf("failed to unmarshal secret YAML: %v", err)
	}

	releaseB64, ok := secret.Data["release"]
	if !ok {
		t.Fatal("secret.Data missing 'release' key")
	}

	// The release field is stored as []byte(base64String), so when marshaled
	// to YAML the bytes are base64-encoded again. We decode once to get the
	// base64-encoded gzip, then decode again to get the gzip bytes.
	gzipB64 := string(releaseB64)
	gzipBytes, err := base64.StdEncoding.DecodeString(gzipB64)
	if err != nil {
		t.Fatalf("failed to base64-decode release: %v", err)
	}

	r, err := gzip.NewReader(bytes.NewReader(gzipBytes))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	jsonBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read gzip data: %v", err)
	}

	var releaseData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &releaseData); err != nil {
		t.Fatalf("failed to unmarshal release JSON: %v", err)
	}

	return &secret, releaseData
}

// findFile finds a BaseFile by path in the given slice.
func findFile(files []BaseFile, path string) *BaseFile {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}

func Test_RenderHelmV4_BasicNamespaceInsertion(t *testing.T) {
	upstream := &upstreamtypes.Upstream{
		Name: "test-chart",
		Files: []upstreamtypes.UpstreamFile{
			{
				Path:    "Chart.yaml",
				Content: []byte("apiVersion: v2\nname: test-chart\nversion: 0.1.0"),
			},
			{
				Path:    "templates/deploy-1.yaml",
				Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1\n  namespace: test-one"),
			},
			{
				Path:    "templates/deploy-2.yaml",
				Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
			},
		},
	}
	opts := &RenderOptions{
		HelmVersion: "v4",
		Namespace:   "test-two",
	}

	got, err := RenderHelm(upstream, opts)
	if err != nil {
		t.Fatalf("RenderHelm() error = %v", err)
	}

	// Verify manifests
	deploy1 := findFile(got.Files, "deploy-1.yaml")
	if deploy1 == nil {
		t.Fatal("deploy-1.yaml not found")
	}
	if !strings.Contains(string(deploy1.Content), "namespace: test-one") {
		t.Errorf("deploy-1.yaml should preserve existing namespace test-one, got:\n%s", deploy1.Content)
	}

	deploy2 := findFile(got.Files, "deploy-2.yaml")
	if deploy2 == nil {
		t.Fatal("deploy-2.yaml not found")
	}
	if !strings.Contains(string(deploy2.Content), "namespace: test-two") {
		t.Errorf("deploy-2.yaml should have namespace test-two injected, got:\n%s", deploy2.Content)
	}

	// Verify the Helm release secret
	secretFile := findFile(got.Files, "chartHelmSecret.yaml")
	if secretFile == nil {
		t.Fatal("chartHelmSecret.yaml not found")
	}

	secret, releaseData := decodeHelmReleaseSecret(t, secretFile.Content)

	// Validate secret structure
	if secret.Type != "helm.sh/release.v1" {
		t.Errorf("secret.Type = %q, want %q", secret.Type, "helm.sh/release.v1")
	}
	if secret.Name != "sh.helm.release.v1.test-chart.v1" {
		t.Errorf("secret.Name = %q, want %q", secret.Name, "sh.helm.release.v1.test-chart.v1")
	}
	if secret.Namespace != "test-two" {
		t.Errorf("secret.Namespace = %q, want %q", secret.Namespace, "test-two")
	}

	// Validate secret labels
	if got := secret.Labels["status"]; got != "deployed" {
		t.Errorf("label status = %q, want %q", got, "deployed")
	}
	if got := secret.Labels["version"]; got != "1" {
		t.Errorf("label version = %q, want %q", got, "1")
	}
	if got := secret.Labels["owner"]; got != "helm" {
		t.Errorf("label owner = %q, want %q", got, "helm")
	}
	if got := secret.Labels["name"]; got != "test-chart" {
		t.Errorf("label name = %q, want %q", got, "test-chart")
	}

	// Validate decoded release data
	if name, _ := releaseData["name"].(string); name != "test-chart" {
		t.Errorf("release.name = %q, want %q", name, "test-chart")
	}
	if ns, _ := releaseData["namespace"].(string); ns != "test-two" {
		t.Errorf("release.namespace = %q, want %q", ns, "test-two")
	}

	// Validate values.yaml is in additional files
	valuesFile := findFile(got.AdditionalFiles, "values.yaml")
	if valuesFile == nil {
		t.Fatal("values.yaml not found in AdditionalFiles")
	}
}

func Test_RenderHelmV4_NoNamespace(t *testing.T) {
	upstream := &upstreamtypes.Upstream{
		Name: "test-chart",
		Files: []upstreamtypes.UpstreamFile{
			{
				Path:    "Chart.yaml",
				Content: []byte("apiVersion: v2\nname: test-chart\nversion: 0.1.0"),
			},
			{
				Path:    "templates/configmap.yaml",
				Content: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: config"),
			},
		},
	}
	opts := &RenderOptions{
		HelmVersion: "v4",
		// No namespace
	}

	got, err := RenderHelm(upstream, opts)
	if err != nil {
		t.Fatalf("RenderHelm() error = %v", err)
	}

	// With no namespace, configmap should not have a namespace added
	cmFile := findFile(got.Files, "configmap.yaml")
	if cmFile == nil {
		t.Fatal("configmap.yaml not found")
	}
	if strings.Contains(string(cmFile.Content), "namespace:") {
		t.Errorf("configmap.yaml should not have namespace, got:\n%s", cmFile.Content)
	}

	// Secret should still be generated but with the kotsadm namespace (from env)
	secretFile := findFile(got.Files, "chartHelmSecret.yaml")
	if secretFile == nil {
		t.Fatal("chartHelmSecret.yaml not found")
	}
	secret, _ := decodeHelmReleaseSecret(t, secretFile.Content)
	if secret.Type != "helm.sh/release.v1" {
		t.Errorf("secret.Type = %q, want %q", secret.Type, "helm.sh/release.v1")
	}
}

func Test_RenderHelmV4_UseHelmInstall_NoSecret(t *testing.T) {
	upstream := &upstreamtypes.Upstream{
		Name: "test-chart",
		Files: []upstreamtypes.UpstreamFile{
			{
				Path:    "Chart.yaml",
				Content: []byte("apiVersion: v2\nname: test-chart\nversion: 0.1.0"),
			},
			{
				Path:    "templates/deploy.yaml",
				Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy"),
			},
		},
	}
	opts := &RenderOptions{
		HelmVersion:    "v4",
		Namespace:      "test-ns",
		UseHelmInstall: true, // native helm install — no secret should be generated
	}

	got, err := RenderHelm(upstream, opts)
	if err != nil {
		t.Fatalf("RenderHelm() error = %v", err)
	}

	// No chartHelmSecret.yaml for native helm installs
	if f := findFile(got.Files, "chartHelmSecret.yaml"); f != nil {
		t.Error("chartHelmSecret.yaml should not be generated for UseHelmInstall=true")
	}
}

func Test_RenderHelmV4_WithValues(t *testing.T) {
	upstream := &upstreamtypes.Upstream{
		Name: "test-chart",
		Files: []upstreamtypes.UpstreamFile{
			{
				Path:    "Chart.yaml",
				Content: []byte("apiVersion: v2\nname: test-chart\nversion: 0.1.0"),
			},
			{
				Path:    "values.yaml",
				Content: []byte("replicaCount: 1\n"),
			},
			{
				Path:    "templates/deploy.yaml",
				Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy\nspec:\n  replicas: {{ .Values.replicaCount }}"),
			},
		},
	}
	opts := &RenderOptions{
		HelmVersion: "v4",
		Namespace:   "test-ns",
		HelmValues:  map[string]interface{}{"replicaCount": 3},
	}

	got, err := RenderHelm(upstream, opts)
	if err != nil {
		t.Fatalf("RenderHelm() error = %v", err)
	}

	deployFile := findFile(got.Files, "deploy.yaml")
	if deployFile == nil {
		t.Fatal("deploy.yaml not found")
	}
	if !strings.Contains(string(deployFile.Content), "replicas: 3") {
		t.Errorf("deploy.yaml should have replicas: 3 from custom values, got:\n%s", deployFile.Content)
	}
}

func Test_RenderHelmV4_SecretCompatibleWithV3(t *testing.T) {
	// Verify that v4-rendered secrets use the same Kubernetes secret format as v3.
	// Both must use type "helm.sh/release.v1" and the same name scheme.
	upstream := &upstreamtypes.Upstream{
		Name: "compat-chart",
		Files: []upstreamtypes.UpstreamFile{
			{
				Path:    "Chart.yaml",
				Content: []byte("apiVersion: v2\nname: compat-chart\nversion: 1.0.0"),
			},
			{
				Path:    "templates/svc.yaml",
				Content: []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: svc"),
			},
		},
	}
	optsV3 := &RenderOptions{HelmVersion: "v3", Namespace: "myns"}
	optsV4 := &RenderOptions{HelmVersion: "v4", Namespace: "myns"}

	gotV3, err := RenderHelm(upstream, optsV3)
	if err != nil {
		t.Fatalf("v3 RenderHelm error: %v", err)
	}
	gotV4, err := RenderHelm(upstream, optsV4)
	if err != nil {
		t.Fatalf("v4 RenderHelm error: %v", err)
	}

	secretV3 := findFile(gotV3.Files, "chartHelmSecret.yaml")
	secretV4 := findFile(gotV4.Files, "chartHelmSecret.yaml")
	if secretV3 == nil || secretV4 == nil {
		t.Fatal("chartHelmSecret.yaml missing from v3 or v4 output")
	}

	var sV3, sV4 v1.Secret
	if err := yaml.Unmarshal(secretV3.Content, &sV3); err != nil {
		t.Fatalf("unmarshal v3 secret: %v", err)
	}
	if err := yaml.Unmarshal(secretV4.Content, &sV4); err != nil {
		t.Fatalf("unmarshal v4 secret: %v", err)
	}

	// Both must use the same secret type
	if sV3.Type != sV4.Type {
		t.Errorf("secret type mismatch: v3=%q v4=%q", sV3.Type, sV4.Type)
	}
	if sV3.Type != "helm.sh/release.v1" {
		t.Errorf("unexpected secret type %q", sV3.Type)
	}

	// Both must use the same naming scheme
	if sV3.Name != sV4.Name {
		t.Errorf("secret name mismatch: v3=%q v4=%q", sV3.Name, sV4.Name)
	}
	expectedName := "sh.helm.release.v1.compat-chart.v1"
	if sV3.Name != expectedName {
		t.Errorf("secret name = %q, want %q", sV3.Name, expectedName)
	}

	// Both must produce correct labels
	for _, s := range []v1.Secret{sV3, sV4} {
		if s.Labels["status"] != "deployed" {
			t.Errorf("label status = %q, want 'deployed'", s.Labels["status"])
		}
		if s.Labels["owner"] != "helm" {
			t.Errorf("label owner = %q, want 'helm'", s.Labels["owner"])
		}
	}
}

func Test_RenderHelmV4_SubchartsWithNamespace(t *testing.T) {
	upstream := &upstreamtypes.Upstream{
		Name: "test-chart",
		Files: []upstreamtypes.UpstreamFile{
			{
				Path:    "Chart.yaml",
				Content: []byte("apiVersion: v2\nname: test-chart\nversion: 0.1.0"),
			},
			{
				Path:    "templates/deploy-1.yaml",
				Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-1"),
			},
			{
				Path:    "charts/test-subchart/Chart.yaml",
				Content: []byte("apiVersion: v2\nname: test-subchart\nversion: 0.2.0"),
			},
			{
				Path:    "charts/test-subchart/templates/deploy-2.yaml",
				Content: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy-2"),
			},
		},
	}
	opts := &RenderOptions{
		HelmVersion:    "v4",
		Namespace:      "test-namespace",
		UseHelmInstall: true,
	}

	got, err := RenderHelm(upstream, opts)
	if err != nil {
		t.Fatalf("RenderHelm() error = %v", err)
	}

	if got.Namespace != "test-namespace" {
		t.Errorf("base.Namespace = %q, want %q", got.Namespace, "test-namespace")
	}

	// Parent chart should have deploy-1
	deploy1 := findFile(got.Files, "templates/deploy-1.yaml")
	if deploy1 == nil {
		t.Fatal("templates/deploy-1.yaml not found in parent chart")
	}
	if !strings.Contains(string(deploy1.Content), "namespace: test-namespace") {
		t.Errorf("deploy-1.yaml should have namespace test-namespace, got:\n%s", deploy1.Content)
	}

	// Subchart should be in Bases
	if len(got.Bases) == 0 {
		t.Fatal("expected subchart in Bases but got none")
	}
	var subchartBase *Base
	for i := range got.Bases {
		if got.Bases[i].Path == "charts/test-subchart" {
			subchartBase = &got.Bases[i]
			break
		}
	}
	if subchartBase == nil {
		t.Fatalf("charts/test-subchart not found in Bases, got: %v", func() []string {
			var paths []string
			for _, b := range got.Bases {
				paths = append(paths, b.Path)
			}
			return paths
		}())
	}

	deploy2 := findFile(subchartBase.Files, "templates/deploy-2.yaml")
	if deploy2 == nil {
		t.Fatal("templates/deploy-2.yaml not found in subchart")
	}
	if !strings.Contains(string(deploy2.Content), "namespace: test-namespace") {
		t.Errorf("deploy-2.yaml should have namespace test-namespace, got:\n%s", deploy2.Content)
	}
}

func Test_RenderHelmV4_VersionSelector(t *testing.T) {
	// Verify that "v4" is recognized and does NOT fall through to an error.
	// Note: helmVersion="v2" uses a legacy Helm 2 renderer that only supports
	// Chart.yaml apiVersion: "v1". We test v4/v3/empty with an apiVersion: v2 chart.
	upstream := &upstreamtypes.Upstream{
		Name: "test",
		Files: []upstreamtypes.UpstreamFile{
			{Path: "Chart.yaml", Content: []byte("apiVersion: v2\nname: test\nversion: 0.1.0")},
		},
	}

	tests := []struct {
		version string
		wantErr bool
	}{
		{"v4", false},
		{"V4", false}, // case-insensitive
		{"v3", false},
		{"", false},  // defaults to v3
		{"v5", true}, // unknown version
		{"v0", true}, // unknown version
	}

	for _, tt := range tests {
		t.Run("helmVersion="+tt.version, func(t *testing.T) {
			opts := &RenderOptions{HelmVersion: tt.version, Namespace: "test-ns"}
			_, err := RenderHelm(upstream, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderHelm(helmVersion=%q) err=%v, wantErr=%v", tt.version, err, tt.wantErr)
			}
		})
	}
}
