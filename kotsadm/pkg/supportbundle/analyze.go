package supportbundle

import (
	"os"

	"github.com/pkg/errors"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
)

func InjectDefaultAnalyzers(analyzer *troubleshootv1beta2.Analyzer) error {
	if err := injectAPIReplicaAnalyzer(analyzer); err != nil {
		return errors.Wrap(err, "failed to inject api replica analyzer")
	}

	if err := injectOperatorReplicaAnalyzer(analyzer); err != nil {
		return errors.Wrap(err, "failed to inject operators replica analyzer")
	}

	if err := injectNoGvisorAnalyzer(analyzer); err != nil {
		return errors.Wrap(err, "failed to inject no gvisor analyzer")
	}

	if err := injectIfMissingKubernetesVersionAnalyzer(analyzer); err != nil {
		return errors.Wrap(err, "failed to inject k8s version analyzer")
	}

	if err := injectCephAnalyzers(analyzer); err != nil {
		return errors.Wrap(err, "failed to inject k8s version analyzer")
	}

	return nil

}

func injectAPIReplicaAnalyzer(analyzer *troubleshootv1beta2.Analyzer) error {
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta2.Analyze{
		DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{
			Name:      "kotsadm",
			Namespace: os.Getenv("POD_NAMESPACE"),
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
	})
	return nil
}

func injectOperatorReplicaAnalyzer(analyzer *troubleshootv1beta2.Analyzer) error {
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta2.Analyze{
		DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{
			Name:      "kotsadm-operator",
			Namespace: os.Getenv("POD_NAMESPACE"),
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						When:    "= 1",
						Message: "Exactly 1 replica of the Admin Console Operator is running and ready",
					},
				},
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						Message: "There is not exactly 1 replica of the Admin Console Operator running and ready",
					},
				},
			},
		},
	})
	return nil
}

func injectNoGvisorAnalyzer(analyzer *troubleshootv1beta2.Analyzer) error {
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta2.Analyze{
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
	})
	return nil
}

func injectIfMissingKubernetesVersionAnalyzer(analyzer *troubleshootv1beta2.Analyzer) error {
	for _, existingAnalyzer := range analyzer.Spec.Analyzers {
		if existingAnalyzer.ClusterVersion != nil {
			return nil
		}
	}

	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta2.Analyze{
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
	})
	return nil
}

func injectCephAnalyzers(analyzer *troubleshootv1beta2.Analyzer) error {
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta2.Analyze{
		CephStatus: &troubleshootv1beta2.CephStatusAnalyze{},
	})
	return nil
}
