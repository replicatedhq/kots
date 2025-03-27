package supportbundle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/redact"
	"github.com/replicatedhq/kots/pkg/store"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootcollect "github.com/replicatedhq/troubleshoot/pkg/collect"
	"github.com/replicatedhq/troubleshoot/pkg/convert"
	troubleshootversion "github.com/replicatedhq/troubleshoot/pkg/version"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateBundleForBackup(appID string, backupName string, backupNamespace string) (string, error) {
	logger.Debug("executing support bundle for backup",
		zap.String("backupName", backupName),
		zap.String("backupNamespace", backupNamespace))

	progressChan := make(chan interface{}, 0)
	defer close(progressChan)

	go func() {
		for {
			msg, ok := <-progressChan
			if ok {
				logger.Debugf("%v", msg)
			} else {
				return
			}
		}
	}()

	var RBACErrors []error

	var collectors []troubleshootcollect.Collector

	restConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get cluster config")
	}

	k8sClientSet, err := k8sutil.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get kubernetes client")
	}

	selectors := []string{
		"component=velero",
		"app.kubernetes.io/name=velero",
	}

	ctx := context.TODO()

	for _, selector := range selectors {
		logsCollector := &troubleshootv1beta2.Logs{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "velero",
			},
			Name:      "velero",
			Namespace: backupNamespace,
			Selector:  []string{selector},
		}

		collectors = append(collectors, &troubleshootcollect.CollectLogs{
			Collector:    logsCollector,
			Namespace:    backupNamespace,
			ClientConfig: restConfig,
			Client:       k8sClientSet,
			Context:      ctx,
			RBACErrors:   RBACErrors,
		})
	}

	// make a temp file to store the bundle in
	bundlePath, err := ioutil.TempDir("", "troubleshoot")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(bundlePath)

	if err = writeVersionFile(bundlePath); err != nil {
		return "", errors.Wrap(err, "failed to write version file")
	}

	redacts := []*troubleshootv1beta2.Redact{}
	globalRedact, err := redact.GetRedact()
	if err == nil && globalRedact != nil {
		redacts = globalRedact.Spec.Redactors
	} else if err != nil {
		return "", errors.Wrap(err, "failed to get global redactors")
	}

	result := make(map[string][]byte)

	// Run preflights collectors synchronously
	for _, collector := range collectors {
		if collector.HasRBACErrors() {
			// don't skip clusterResources collector due to RBAC issues
			if _, ok := collector.(*troubleshootcollect.CollectClusterResources); !ok {
				progressChan <- fmt.Sprintf("skipping collector %s with insufficient RBAC permissions", collector.Title())
				continue
			}
		}

		progressChan <- collector.Title()

		result, err = collector.Collect(progressChan)
		if err != nil {
			progressChan <- errors.Wrapf(err, "failed to run collector %q", collector.Title())
			continue
		}

		if result != nil {
			err = saveCollectorOutput(result, bundlePath)
			if err != nil {
				progressChan <- errors.Wrapf(err, "failed to parse collector spec %q", collector.Title())
				continue
			}
		}
	}

	// Redact result before creating archive
	err = troubleshootcollect.RedactResult(bundlePath, result, redacts)
	if err != nil {
		return "", errors.Wrap(err, "failed to redact")
	}

	// create an archive of this bundle
	supportBundleArchivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create archive dir")
	}
	defer os.RemoveAll(supportBundleArchivePath)

	if err = tarSupportBundleDir(bundlePath, filepath.Join(supportBundleArchivePath, "support-bundle.tar.gz")); err != nil {
		return "", errors.Wrap(err, "failed to create support bundle archive")
	}

	// we have a support bundle...
	// store it
	supportBundle, err := CreateBundle(
		fmt.Sprintf("backup-%s", backupName),
		appID,
		filepath.Join(supportBundleArchivePath, "support-bundle.tar.gz"))
	if err != nil {
		return "", errors.Wrap(err, "failed to create support bundle")
	}

	// analyze it
	analyzers := []*troubleshootv1beta2.Analyze{}

	analyzers = append(analyzers, &troubleshootv1beta2.Analyze{
		TextAnalyze: &troubleshootv1beta2.TextAnalyze{
			AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
				CheckName: "Velero Errors",
			},
			CollectorName: "velero",
			FileName:      "velero/velero*/velero.log",
			RegexPattern:  "level=error",
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						Message: "Velero has errors",
					},
				},
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						Message: "Velero does not have errors",
					},
				},
			},
		},
	})
	analyzers = append(analyzers, &troubleshootv1beta2.Analyze{
		TextAnalyze: &troubleshootv1beta2.TextAnalyze{
			AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
				CheckName: "Restic Volumes",
			},
			CollectorName: "restic",
			FileName:      "restic/*.log",
			RegexPattern:  "expected one matching path, got 0",
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						Message: "Restic volume error",
					},
				},
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						Message: "No restic volume error",
					},
				},
			},
		},
	})

	analyzer := troubleshootv1beta2.Analyzer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "troubleshoot.sh/v1beta2",
			Kind:       "Analyzer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: backupName,
		},
		Spec: troubleshootv1beta2.AnalyzerSpec{
			Analyzers: analyzers,
		},
	}
	b, err := json.Marshal(analyzer)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal analyzers")
	}

	analyzeResult, err := troubleshootanalyze.DownloadAndAnalyze(filepath.Join(supportBundleArchivePath, "support-bundle.tar.gz"), string(b))
	if err != nil {
		return "", errors.Wrap(err, "failed to analyze")
	}

	data := convert.FromAnalyzerResult(analyzeResult)
	insights, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal analysis")
	}

	if err := store.GetStore().SetSupportBundleAnalysis(supportBundle.ID, insights); err != nil {
		return "", errors.Wrap(err, "failed to update bundle status")
	}
	return supportBundle.ID, nil
}

func tarSupportBundleDir(inputDir string, outputFilename string) error {
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
			OverwriteExisting:      true,
		},
	}

	paths := []string{
		filepath.Join(inputDir, "version.yaml"), // version file should be first in tar archive for quick extraction
	}

	topLevelFiles, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return errors.Wrap(err, "failed to list bundle directory contents")
	}
	for _, f := range topLevelFiles {
		if f.Name() == "version.yaml" {
			continue
		}
		paths = append(paths, filepath.Join(inputDir, f.Name()))
	}

	if err := tarGz.Archive(paths, outputFilename); err != nil {
		return errors.Wrap(err, "failed to create archive")
	}

	return nil
}

func saveCollectorOutput(output map[string][]byte, bundlePath string) error {
	for filename, maybeContents := range output {
		fileDir, fileName := filepath.Split(filename)
		outPath := filepath.Join(bundlePath, fileDir)

		if err := os.MkdirAll(outPath, 0777); err != nil {
			return errors.Wrap(err, "failed to create output file")
		}

		if err := ioutil.WriteFile(filepath.Join(outPath, fileName), maybeContents, 0644); err != nil {
			return errors.Wrap(err, "failed to write file")
		}
	}

	return nil
}

func writeVersionFile(path string) error {
	version := troubleshootv1beta2.SupportBundleVersion{
		ApiVersion: "troubleshoot.sh/v1beta2",
		Kind:       "SupportBundle",
		Spec: troubleshootv1beta2.SupportBundleVersionSpec{
			VersionNumber: troubleshootversion.Version(),
		},
	}
	b, err := yaml.Marshal(version)
	if err != nil {
		return err
	}

	filename := filepath.Join(path, "version.yaml")
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		return err
	}

	return nil
}
