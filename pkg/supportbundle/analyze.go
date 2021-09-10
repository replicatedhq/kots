package supportbundle

import (
	"os"

	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
)

func InjectDefaultAnalyzers(analyzers []*troubleshootv1beta2.Analyze) []*troubleshootv1beta2.Analyze {
	analyzers = append(analyzers, getAPIReplicaAnalyzer())
	analyzers = append(analyzers, getNoGvisorAnalyzer())
	analyzers = append(analyzers, getIfMissingKubernetesVersionAnalyzer(analyzers))
	analyzers = append(analyzers, getCephAnalyzers())
	analyzers = append(analyzers, getLonghornAnalyzers())
	analyzers = append(analyzers, getWeaveReportAnalyzer())
	return analyzers
}

func getAPIReplicaAnalyzer() *troubleshootv1beta2.Analyze {
	if os.Getenv("POD_OWNER_KIND") == "deployment" {
		return &troubleshootv1beta2.Analyze{
			DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{
				Name:      "kotsadm",
				Namespace: util.PodNamespace,
				Outcomes: []*troubleshootv1beta2.Outcome{
					{
						Pass: &troubleshootv1beta2.SingleOutcome{
							When:    "> 0",
							Message: "At least 1 replica of the Admin Console API is running and ready",
						},
					},
					{
						Fail: &troubleshootv1beta2.SingleOutcome{
							When:    "= 0",
							Message: "There are no replicas of the Admin Console API running and ready",
						},
					},
				},
			},
		}
	}
	return &troubleshootv1beta2.Analyze{
		StatefulsetStatus: &troubleshootv1beta2.StatefulsetStatus{
			Name:      "kotsadm",
			Namespace: util.PodNamespace,
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						When:    "> 0",
						Message: "At least 1 replica of the Admin Console API is running and ready",
					},
				},
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						When:    "= 0",
						Message: "There are no replicas of the Admin Console API running and ready",
					},
				},
			},
		},
	}
}

func getNoGvisorAnalyzer() *troubleshootv1beta2.Analyze {
	return &troubleshootv1beta2.Analyze{
		ContainerRuntime: &troubleshootv1beta2.ContainerRuntime{
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						When:    "== gvisor",
						Message: "The Admin Console does not support using the gvisor runtime",
					},
				},
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						Message: "A supported container runtime is present on all nodes",
					},
				},
			},
		},
	}
}

func getIfMissingKubernetesVersionAnalyzer(analyzers []*troubleshootv1beta2.Analyze) *troubleshootv1beta2.Analyze {
	for _, existingAnalyzer := range analyzers {
		if existingAnalyzer.ClusterVersion != nil {
			return nil
		}
	}
	return &troubleshootv1beta2.Analyze{
		ClusterVersion: &troubleshootv1beta2.ClusterVersion{
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						When:    "< 1.16.0",
						Message: "The Admin Console requires at least Kubernetes 1.16.0",
					},
				},
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						Message: "Your cluster meets the recommended and required versions of Kubernetes",
					},
				},
			},
		},
	}
}

func getCephAnalyzers() *troubleshootv1beta2.Analyze {
	return &troubleshootv1beta2.Analyze{
		CephStatus: &troubleshootv1beta2.CephStatusAnalyze{},
	}
}

func getLonghornAnalyzers() *troubleshootv1beta2.Analyze {
	return &troubleshootv1beta2.Analyze{
		Longhorn: &troubleshootv1beta2.LonghornAnalyze{},
	}
}

func getWeaveReportAnalyzer() *troubleshootv1beta2.Analyze {
	return &troubleshootv1beta2.Analyze{
		WeaveReport: &troubleshootv1beta2.WeaveReportAnalyze{
			ReportFileGlob: "kots/kurl/weave/kube-system/*/weave-report-stdout.txt",
		},
	}
}
