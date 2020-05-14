package v1beta1tests

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
	"k8s.io/client-go/kubernetes/scheme"
)

func Test_HelmChart(t *testing.T) {
	data := `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: test
spec:
  # helmVersion identifies the Helm Version used to render the Chart. Default is v2.
  helmVersion: v3

  # chart identifies a matching chart from a .tgz
  chart:
    name: test
    chartVersion: 1.5.0-beta.2

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values:
    secretKey: '{{repl ConfigOption "secret_key"}}'
    components:
      worker:
        replicaCount: repl{{ ConfigOption "worker_replica_count"}}

    externalCacheDSN: repl{{ if ConfigOptionEquals "redis_location" "redis_location_external"}}{{repl ConfigOption "external_redis_dsn"}}{{repl end}}

    ingress:
      enabled: repl{{ ConfigOptionEquals "ingress_enabled" "ingress_enabled_yes"}}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder:
    ingress:
      enabled: true
      test: ~
      1: 4
      a: 4.0
      b: ["a", "b"]
`

	kotsscheme.AddToScheme(scheme.Scheme)

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "kots.io", gvk.Group)
	assert.Equal(t, "v1beta1", gvk.Version)
	assert.Equal(t, "HelmChart", gvk.Kind)

	helmChart := obj.(*kotsv1beta1.HelmChart)

	assert.Equal(t, "v3", helmChart.Spec.HelmVersion)

	assert.Equal(t, "test", helmChart.Spec.Chart.Name)
	assert.Equal(t, "1.5.0-beta.2", helmChart.Spec.Chart.ChartVersion)

}
