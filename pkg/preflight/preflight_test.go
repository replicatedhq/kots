package preflight

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/preflight/types"
	kurlv1beta1 "github.com/replicatedhq/kurlkinds/pkg/apis/cluster/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_GetPreflightState(t *testing.T) {
	tests := []struct {
		name             string
		preflightResults *types.PreflightResults
		ignoreNonStrict  bool
		want             string
	}{
		{
			name: "pass",
			preflightResults: &types.PreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{},
					{},
				},
			},
			want: "pass",
		},
		{
			name: "warn",
			preflightResults: &types.PreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{IsWarn: true},
					{},
				},
			},
			want: "warn",
		},
		{
			name: "fail",
			preflightResults: &types.PreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{IsFail: true},
					{},
				},
			},
			want: "fail",
		},
		{
			name: "pass ignoring non-strict preflights",
			preflightResults: &types.PreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{IsFail: true},
					{},
				},
			},
			ignoreNonStrict: true,
			want:            "pass",
		},
		{
			name: "fail ignoring non-strict preflights",
			preflightResults: &types.PreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{Strict: true, IsFail: true},
					{},
				},
			},
			ignoreNonStrict: true,
			want:            "fail",
		},
		{
			name: "error",
			preflightResults: &types.PreflightResults{
				Results: []*troubleshootpreflight.UploadPreflightResult{
					{},
					{IsWarn: true},
					{},
				},
				Errors: []*types.PreflightError{
					{},
				},
			},
			want: "fail",
		},
		{
			name:             "empty",
			preflightResults: &types.PreflightResults{},
			want:             "pass",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := GetPreflightState(test.preflightResults, test.ignoreNonStrict); got != test.want {
				t.Errorf("GetPreflightState() = %v, want %v", got, test.want)
			}
		})
	}
}

func Test_injectInstallerPreflightIfPresent(t *testing.T) {
	tests := []struct {
		name              string
		preflight         *troubleshootv1beta2.Preflight
		deployedInstaller *kurlv1beta1.Installer
		releaseInstaller  *kurlv1beta1.Installer
		want              bool
	}{
		{
			name: "installer-preflight-is-injected",
			preflight: &troubleshootv1beta2.Preflight{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "troubleshoot.sh/v1beta2",
					Kind:       "Preflight",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "preflight-test",
				},
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							YamlCompare: &troubleshootv1beta2.YamlCompare{
								AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
									CheckName:   "Kubernetes Installer",
									Annotations: map[string]string{"kots.io/installer": "true"},
								},
								Outcomes: []*troubleshootv1beta2.Outcome{
									{
										Fail: &troubleshootv1beta2.SingleOutcome{
											Message: "The Kubernetes installer for this version is different from what you have installed.",
											URI:     "https://kurl.sh",
										},
									},
									{
										Pass: &troubleshootv1beta2.SingleOutcome{
											Message: "The Kubernetes installer for this version matches what you have installed.",
										},
									},
								},
							},
						},
					},
				},
			},
			deployedInstaller: &kurlv1beta1.Installer{
				Spec: kurlv1beta1.InstallerSpec{
					Kubernetes: &kurlv1beta1.Kubernetes{
						Version: "1.23.6",
					},
					Containerd: &kurlv1beta1.Containerd{
						Version: "1.5.11",
					},
				},
			},
			releaseInstaller: &kurlv1beta1.Installer{
				Spec: kurlv1beta1.InstallerSpec{
					Kubernetes: &kurlv1beta1.Kubernetes{
						Version: "1.23.6",
					},
					Containerd: &kurlv1beta1.Containerd{
						Version: "1.5.11",
					},
				},
			},
			want: true,
		},
		{
			name: "installer-preflight-is-not-injected-no-annotation",
			preflight: &troubleshootv1beta2.Preflight{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "troubleshoot.sh/v1beta2",
					Kind:       "Preflight",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "preflight-test",
				},
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{
						{
							YamlCompare: &troubleshootv1beta2.YamlCompare{
								AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
									CheckName: "Kubernetes Installer",
								},
								Outcomes: []*troubleshootv1beta2.Outcome{
									{
										Fail: &troubleshootv1beta2.SingleOutcome{
											Message: "The Kubernetes installer for this version is different from what you have installed.",
											URI:     "https://kurl.sh",
										},
									},
									{
										Pass: &troubleshootv1beta2.SingleOutcome{
											Message: "The Kubernetes installer for this version matches what you have installed.",
										},
									},
								},
							},
						},
					},
				},
			},
			deployedInstaller: &kurlv1beta1.Installer{
				Spec: kurlv1beta1.InstallerSpec{
					Kubernetes: &kurlv1beta1.Kubernetes{
						Version: "1.23.6",
					},
					Containerd: &kurlv1beta1.Containerd{
						Version: "1.5.11",
					},
				},
			},
			releaseInstaller: &kurlv1beta1.Installer{
				Spec: kurlv1beta1.InstallerSpec{
					Kubernetes: &kurlv1beta1.Kubernetes{
						Version: "1.23.6",
					},
					Containerd: &kurlv1beta1.Containerd{
						Version: "1.5.11",
					},
				},
			},
			want: false,
		},
		{
			name: "installer-preflight-is-not-injected-no-analyzer",
			preflight: &troubleshootv1beta2.Preflight{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "troubleshoot.sh/v1beta2",
					Kind:       "Preflight",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "preflight-test",
				},
				Spec: troubleshootv1beta2.PreflightSpec{
					Analyzers: []*troubleshootv1beta2.Analyze{},
				},
			},
			deployedInstaller: &kurlv1beta1.Installer{
				Spec: kurlv1beta1.InstallerSpec{
					Kubernetes: &kurlv1beta1.Kubernetes{
						Version: "1.23.6",
					},
					Containerd: &kurlv1beta1.Containerd{
						Version: "1.5.11",
					},
				},
			},
			releaseInstaller: &kurlv1beta1.Installer{
				Spec: kurlv1beta1.InstallerSpec{
					Kubernetes: &kurlv1beta1.Kubernetes{
						Version: "1.23.6",
					},
					Containerd: &kurlv1beta1.Containerd{
						Version: "1.5.11",
					},
				},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			injectInstallerPreflightIfPresent(test.preflight, test.deployedInstaller, test.releaseInstaller)

			deployedInstallerSpecYaml, err := yaml.Marshal(test.deployedInstaller.Spec)
			req.NoError(err)

			releaseInstallerSpecYaml, err := yaml.Marshal(test.releaseInstaller.Spec)
			req.NoError(err)

			injectedDeployedSpec := len(test.preflight.Spec.Collectors) > 0 && test.preflight.Spec.Collectors[0].Data.Data == string(deployedInstallerSpecYaml)
			injectedReleaseSpec := len(test.preflight.Spec.Analyzers) > 0 && test.preflight.Spec.Analyzers[0].YamlCompare.Value == string(releaseInstallerSpecYaml)

			injected := injectedDeployedSpec && injectedReleaseSpec

			if injected != test.want {
				t.Errorf("installer preflight injected = %v, want %v", injected, test.want)
			}
		})
	}
}
