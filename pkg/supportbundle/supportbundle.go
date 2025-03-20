package supportbundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/redact"
	"github.com/replicatedhq/kots/pkg/render/helper"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/convert"
	"github.com/segmentio/ksuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

const (
	SpecDataKey = "support-bundle-spec"
)

// Collect will queue collection of a new support bundle.
// It returns the ID of the support bundle so that the status can be queried by the
// front end.
func Collect(app *apptypes.App, clusterID string) (string, error) {
	sequence := int64(0)

	currentVersion, err := store.GetStore().GetCurrentDownstreamVersion(app.ID, clusterID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get current downstream version")
	}
	if currentVersion != nil {
		sequence = currentVersion.Sequence
	}

	opts := types.TroubleshootOptions{
		DisableUpload: true,
	}
	supportBundle, err := CreateSupportBundleDependencies(app, sequence, opts)
	if err != nil {
		return "", errors.Wrap(err, "could not generate support bundle dependencies")
	}

	supportBundle.ID = strings.ToLower(ksuid.New().String())
	supportBundle.Slug = supportBundle.ID

	err = store.GetStore().CreateInProgressSupportBundle(supportBundle)
	if err != nil {
		return "", errors.Wrap(err, "could not generate support bundle in progress")
	}

	progressChan := executeUpdateRoutine(supportBundle)
	executeSupportBundleCollectRoutine(supportBundle, progressChan)

	return supportBundle.ID, nil
}

// CreateBundle will create a support bundle in the store, attempting to use the
// requestedID. This function uploads the archive and creates the record.
func CreateBundle(requestedID string, appID string, archivePath string) (*types.SupportBundle, error) {
	id := ksuid.New().String()
	if requestedID != "" {
		id = requestedID
	}

	fileTree, err := archiveToFileTree(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate file tree")
	}

	marshalledTree, err := json.Marshal(fileTree.Nodes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal tree index")
	}

	return store.GetStore().CreateSupportBundle(id, appID, archivePath, marshalledTree)
}

func GetSpecName(appSlug string) string {
	return fmt.Sprintf("kotsadm-%s-supportbundle", appSlug)
}

func GetSpecURI(appSlug string) string {
	return fmt.Sprintf("secret/%s/%s", util.PodNamespace, GetSpecName(appSlug))
}

func GetBundleCommand(appSlug string) []string {
	redactURIs := []string{redact.GetKotsadmRedactSpecURI(), redact.GetAppRedactSpecURI(appSlug)}
	redactors := strings.Join(redactURIs, ",")

	command := []string{
		"curl https://krew.sh/support-bundle | bash",
		fmt.Sprintf("kubectl support-bundle --load-cluster-specs --redactors=%s\n", redactors),
	}

	return command
}

// CreateSupportBundleDependencies generates k8s secrets and configmaps for the support bundle spec and redactors.
// These resources will be used when executing a support bundle collection
func CreateSupportBundleDependencies(app *apptypes.App, sequence int64, opts types.TroubleshootOptions) (*types.SupportBundle, error) {
	kotsKinds, err := getKotsKindsForApp(app, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kots kinds for app")
	}

	supportBundle, err := CreateRenderedSpec(app, sequence, kotsKinds, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	// redactors configured in the admin console (from kotsadm-redact backend and written to kotsadm-redact-spec)
	err = redact.GenerateKotsadmRedactSpec(clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write kotsadm redact spec configmap")
	}
	redactURIs := []string{redact.GetKotsadmRedactSpecURI()}

	// redactors configured in the app spec (written to kotsadm-<app-slug>-redact-spec)
	err = redact.CreateRenderedAppRedactSpec(clientset, app, sequence, kotsKinds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write app redact spec configmap")
	}
	redactURIs = append(redactURIs, redact.GetAppRedactSpecURI(app.GetSlug()))

	// default redactors applied to all support bundles (written to kotsadm-redact-default-spec)
	err = redact.CreateRenderedDefaultRedactSpec(clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write default redact spec configmap")
	}
	redactURIs = append(redactURIs, redact.GetDefaultRedactSpecURI())

	supportBundleObj := types.SupportBundle{
		AppID:      app.GetID(),
		URI:        GetSpecURI(app.GetSlug()),
		RedactURIs: redactURIs,
		Progress: types.SupportBundleProgress{
			CollectorCount: len(supportBundle.Spec.Collectors),
		},
	}

	return &supportBundleObj, nil
}

func getKotsKindsForApp(app *apptypes.App, sequence int64) (*kotsutil.KotsKinds, error) {
	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(app.GetID(), sequence, archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load current kotskinds")
	}

	return kotsKinds, nil
}

func getAnalysisFromBundle(archivePath string) ([]byte, error) {
	bundleDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(bundleDir)

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(archivePath, bundleDir); err != nil {
		return nil, errors.Wrap(err, "failed to unarchive")
	}

	var analysis []byte
	err = filepath.Walk(bundleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// trim the directory and subdirectory from the path
		// for example: "support-bundle-2021-09-10T18_50_35/support-bundle-2021-09-10T18_50_35/analysis.json"
		relPath, err := filepath.Rel(bundleDir, path) // becomes: "support-bundle-2021-09-10T18_50_35/analysis.json"
		if err != nil {
			return errors.Wrap(err, "failed to get relative path")
		}
		trimmedRelPath := SupportBundleNameRegex.ReplaceAllString(relPath, "")        // becomes: "analysis.json"
		trimmedRelPath = strings.TrimPrefix(trimmedRelPath, string(os.PathSeparator)) // extra measure to ensure no leading slashes. for example: "/analysis.json"

		if trimmedRelPath == "analysis.json" {
			b, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read analysis file")
			}
			analysis = b
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk")
	}

	return analysis, nil
}

// CreateSupportBundleAnalysis adds the analysis to the support bundle secret.
// if the support bundle archive already includes analysis, the secret will be updated with that. (which is the case for new support bundles)
// if not, the support bundle archive will be unpacked and analyzed and then the secret will be updated with the results. (which is the case for older support bundles formats)
func CreateSupportBundleAnalysis(appID string, archivePath string, bundle *types.SupportBundle) error {
	analysis, err := getAnalysisFromBundle(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to check if support bundle includes analysis")
	}
	if len(analysis) > 0 {
		// new support bundles include the analysis as part of the bundle
		if err := store.GetStore().SetSupportBundleAnalysis(bundle.ID, analysis); err != nil {
			return errors.Wrap(err, "failed to set support bundle analysis")
		}
		return nil
	}

	// we need the app archive to get the analyzers
	foundApp, err := store.GetStore().GetApp(appID)
	if err != nil {
		err = errors.Wrap(err, "failed to get app")
		logger.Error(err)
		return err
	}

	latestSequence, err := store.GetStore().GetLatestAppSequence(foundApp.ID, true)
	if err != nil {
		return errors.Wrap(err, "failed to get latest app sequence")
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		err = errors.Wrap(err, "failed to create temp dir")
		logger.Error(err)
		return err
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, latestSequence, archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to get app version archive")
		logger.Error(err)
		return err
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to load kots kinds from archive")
		logger.Error(err)
		return err
	}

	analyzer := kotsKinds.Analyzer
	// SupportBundle overwrites Analyzer if defined
	if kotsKinds.SupportBundle != nil {
		analyzer = kotsutil.SupportBundleToAnalyzer(kotsKinds.SupportBundle)
	}
	if analyzer == nil {
		analyzer = &troubleshootv1beta2.Analyzer{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "troubleshoot.sh/v1beta2",
				Kind:       "Analyzer",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-analyzers",
			},
			Spec: troubleshootv1beta2.AnalyzerSpec{
				Analyzers: []*troubleshootv1beta2.Analyze{},
			},
		}
	}
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Errorf("Failed to get kubernetes clientset: %v", err)
	}
	isKurl, err := kurl.IsKurl(clientset)
	if err != nil {
		logger.Errorf("Failed to check if cluster is kurl: %v", err)
	}

	defaultAnalyzers, err := getDefaultAnalyzers(isKurl, foundApp)
	if err != nil {
		return errors.Wrap(err, "failed to get default analyzers")
	}
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, defaultAnalyzers...)
	analyzer.Spec.Analyzers = append(analyzer.Spec.Analyzers, getDefaultDynamicAnalyzers(foundApp)...)

	s := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(analyzer, &b); err != nil {
		err = errors.Wrap(err, "failed to encode analyzers")
		logger.Error(err)
		return err
	}

	renderedAnalyzers, err := helper.RenderAppFile(foundApp, nil, b.Bytes(), kotsKinds, util.PodNamespace)
	if err != nil {
		err = errors.Wrap(err, "failed to render analyzers")
		logger.Error(err)
		return err
	}

	analyzeResult, err := troubleshootanalyze.DownloadAndAnalyze(archivePath, string(renderedAnalyzers))
	if err != nil {
		err = errors.Wrap(err, "failed to analyze")
		logger.Error(err)
		return err
	}

	data := convert.FromAnalyzerResult(analyzeResult)
	insights, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		err = errors.Wrap(err, "failed to marshal result")
		logger.Error(err)
		return err
	}

	if err := store.GetStore().SetSupportBundleAnalysis(bundle.ID, insights); err != nil {
		err = errors.Wrap(err, "failed to save result")
		logger.Error(err)
		return err
	}

	return nil
}
