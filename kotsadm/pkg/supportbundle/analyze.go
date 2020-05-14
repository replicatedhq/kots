package supportbundle

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	troubleshootv1beta1 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta1"
	"github.com/segmentio/ksuid"
)

func SetBundleAnalysis(id string, insights []byte) error {
	db := persistence.MustGetPGSession()
	query := `update supportbundle set status = $1 where id = $2`

	_, err := db.Exec(query, "analyzed", id)
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}

	query = `insert into supportbundle_analysis (id, supportbundle_id, error, max_severity, insights, created_at) values ($1, $2, null, null, $3, $4)`
	_, err = db.Exec(query, ksuid.New().String(), id, insights, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to insert insights")
	}

	return nil
}

func InjectDefaultAnalyzers(analyzer *troubleshootv1beta1.Analyzer) error {
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

	return nil

}

func injectAPIReplicaAnalyzer(analyzer *troubleshootv1beta1.Analyzer) error {
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta1.Analyze{
		DeploymentStatus: &troubleshootv1beta1.DeploymentStatus{
			Name:      "kotsadm-api",
			Namespace: os.Getenv("POD_NAMESPACE"),
			Outcomes: []*troubleshootv1beta1.Outcome{
				{
					Pass: &troubleshootv1beta1.SingleOutcome{
						When:    "> 1",
						Message: "At least 2 replicas of the Admin Console API is running and ready",
					},
					Warn: &troubleshootv1beta1.SingleOutcome{
						When:    "= 1",
						Message: "Only 1 replica of the Admin Console API is running and ready",
					},
					Fail: &troubleshootv1beta1.SingleOutcome{
						When:    "= 0",
						Message: "There are no replicas of the Admin Console API running and ready",
					},
				},
			},
		},
	})
	return nil
}

func injectOperatorReplicaAnalyzer(analyzer *troubleshootv1beta1.Analyzer) error {
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta1.Analyze{
		DeploymentStatus: &troubleshootv1beta1.DeploymentStatus{
			Name:      "kotsadm-operator",
			Namespace: os.Getenv("POD_NAMESPACE"),
			Outcomes: []*troubleshootv1beta1.Outcome{
				{
					Pass: &troubleshootv1beta1.SingleOutcome{
						When:    "= 1",
						Message: "Exactly 1 replica of the Admin Console Operator is running and ready",
					},
					Fail: &troubleshootv1beta1.SingleOutcome{
						Message: "There is not exactly 1 replica of the Admin Console Operator running and ready",
					},
				},
			},
		},
	})
	return nil
}

func injectNoGvisorAnalyzer(analyzer *troubleshootv1beta1.Analyzer) error {
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta1.Analyze{
		ContainerRuntime: &troubleshootv1beta1.ContainerRuntime{
			Outcomes: []*troubleshootv1beta1.Outcome{
				{
					Fail: &troubleshootv1beta1.SingleOutcome{
						When:    "== gvisor",
						Message: "The Admin Console does not support using the gvisor runtime",
					},
					Pass: &troubleshootv1beta1.SingleOutcome{
						Message: "A supported container runtime is present on all nodes",
					},
				},
			},
		},
	})
	return nil
}

func injectIfMissingKubernetesVersionAnalyzer(analyzer *troubleshootv1beta1.Analyzer) error {
	for _, existingAnalyzer := range analyzer.Spec.Analyzers {
		if existingAnalyzer.ClusterVersion != nil {
			return nil
		}
	}

	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, &troubleshootv1beta1.Analyze{
		ClusterVersion: &troubleshootv1beta1.ClusterVersion{
			Outcomes: []*troubleshootv1beta1.Outcome{
				{
					Fail: &troubleshootv1beta1.SingleOutcome{
						When:    "< 1.16.0",
						Message: "The Admin Console requires at least Kubernetes 1.16.0",
					},
					Pass: &troubleshootv1beta1.SingleOutcome{
						Message: "Your cluster meets the recommended and required versions of Kubernetes",
					},
				},
			},
		},
	})
	return nil
}
