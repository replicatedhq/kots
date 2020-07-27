package supportbundle

import (
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
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

func GetBundleAnalysis(id string) (*types.SupportBundleAnalysis, error) {
	db := persistence.MustGetPGSession()
	query := `SELECT id, error, max_severity, insights, created_at FROM supportbundle_analysis where supportbundle_id = $1`
	row := db.QueryRow(query, id)

	var _error sql.NullString
	var maxSeverity sql.NullString
	var insightsStr sql.NullString

	a := &types.SupportBundleAnalysis{}
	if err := row.Scan(&a.ID, &_error, &maxSeverity, &insightsStr, &a.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	a.Error = _error.String
	a.MaxSeverity = maxSeverity.String

	if insightsStr.Valid {
		type Insight struct {
			Primary string `json:"primary"`
			Detail  string `json:"detail"`
		}
		type Labels struct {
			IconUri         string `json:"iconUri"`
			IconKey         string `json:"iconKey"`
			DesiredPosition string `json:"desiredPosition"`
		}
		type DBInsight struct {
			Name     string  `json:"name"`
			Severity string  `json:"severity"`
			Insight  Insight `json:"insight"`
			Labels   Labels  `json:"labels"`
		}

		dbInsights := []DBInsight{}
		if err := json.Unmarshal([]byte(insightsStr.String), &dbInsights); err != nil {
			logger.Error(errors.Wrap(err, "failed to unmarshal db insights"))
			dbInsights = []DBInsight{}
		}

		insights := []types.SupportBundleInsight{}
		for _, dbInsight := range dbInsights {
			desiredPosition, _ := strconv.ParseFloat(dbInsight.Labels.DesiredPosition, 64)
			insight := types.SupportBundleInsight{
				Key:             dbInsight.Name,
				Severity:        dbInsight.Severity,
				Primary:         dbInsight.Insight.Primary,
				Detail:          dbInsight.Insight.Detail,
				Icon:            dbInsight.Labels.IconUri,
				IconKey:         dbInsight.Labels.IconKey,
				DesiredPosition: desiredPosition,
			}
			insights = append(insights, insight)
		}

		a.Insights = insights
	}

	return a, nil
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
				},
				{
					Warn: &troubleshootv1beta1.SingleOutcome{
						When:    "= 1",
						Message: "Only 1 replica of the Admin Console API is running and ready",
					},
				},
				{
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
				},
				{
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
				},
				{
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
				},
				{
					Pass: &troubleshootv1beta1.SingleOutcome{
						Message: "Your cluster meets the recommended and required versions of Kubernetes",
					},
				},
			},
		},
	})
	return nil
}
