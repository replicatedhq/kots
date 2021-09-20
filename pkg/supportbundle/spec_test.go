package supportbundle

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/require"
)

func TestBuilder_populateNamespaces(t *testing.T) {
	origPodNamespace := util.PodNamespace
	util.PodNamespace = "populateNamespaces"
	defer func() {
		util.PodNamespace = origPodNamespace
	}()

	tests := []struct {
		name          string
		supportBundle troubleshootv1beta2.SupportBundle
		want          troubleshootv1beta2.SupportBundle
	}{
		{
			name: "all",
			supportBundle: troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							Secret: &troubleshootv1beta2.Secret{
								Namespace: `repl{{ ConfigOption "test" }}`,
							},
							Run: &troubleshootv1beta2.Run{
								Namespace: util.PodNamespace,
							},
							Logs: &troubleshootv1beta2.Logs{
								Namespace: "hardcoded",
							},
							Exec: &troubleshootv1beta2.Exec{
								Namespace: "",
							},
							Copy: &troubleshootv1beta2.Copy{
								Namespace: `repl{{ Namespace }}`,
							},
						},
					},
				},
			},
			want: troubleshootv1beta2.SupportBundle{
				Spec: troubleshootv1beta2.SupportBundleSpec{
					Collectors: []*troubleshootv1beta2.Collect{
						{
							Secret: &troubleshootv1beta2.Secret{
								Namespace: `repl{{ ConfigOption "test" }}`,
							},
							Run: &troubleshootv1beta2.Run{
								Namespace: util.PodNamespace,
							},
							Logs: &troubleshootv1beta2.Logs{
								Namespace: "hardcoded",
							},
							Exec: &troubleshootv1beta2.Exec{
								Namespace: util.PodNamespace,
							},
							Copy: &troubleshootv1beta2.Copy{
								Namespace: `repl{{ Namespace }}`,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			populateNamespaces(&tt.supportBundle)

			req.Equal(tt.want, tt.supportBundle)
		})
	}
}
