package supportbundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/redact"
	"github.com/replicatedhq/kots/pkg/render/helper"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
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
func Collect(appID string, clusterID string) (string, error) {
	sequence := int64(0)

	currentVersion, err := store.GetStore().GetCurrentVersion(appID, clusterID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get current downstream version")
	}
	if currentVersion != nil {
		sequence = currentVersion.Sequence
	}

	opts := types.TroubleshootOptions{
		DisableUpload: true,
	}
	supportBundle, err := CreateSupportBundleDependencies(appID, sequence, opts)
	if err != nil {
		return "", errors.Wrap(err, "could not generate support bundle dependencies")
	}

	supportBundle.ID = strings.ToLower(ksuid.New().String())
	supportBundle.Slug = supportBundle.ID

	store.GetStore().CreateInProgressSupportBundle(supportBundle)

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

// GetFilesContents will return the file contents for filenames matching the filenames
// parameter.
func GetFilesContents(bundleID string, filenames []string) (map[string][]byte, error) {
	bundleArchive, err := store.GetStore().GetSupportBundleArchive(bundleID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bundle")
	}
	defer os.RemoveAll(bundleArchive)

	bundleDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tmp dir")
	}
	defer os.RemoveAll(bundleDir)

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(bundleArchive, bundleDir); err != nil {
		return nil, errors.Wrap(err, "failed to unarchive")
	}

	files := map[string][]byte{}
	err = filepath.Walk(bundleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if len(path) <= len(bundleDir) {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// the following tries to find the actual file path of the desired files in the support bundle
		// this is needed to handle old and new support bundle formats
		// where old support bundles don't include a top level subdirectory and the new ones do
		// this basically compares file paths after trimming the subdirectory path from both (if exists)
		// for example: "support-bundle-2021-09-10T18_50_35/support-bundle-2021-09-10T18_50_35/path/to/file"
		relPath, err := filepath.Rel(bundleDir, path) // becomes: "support-bundle-2021-09-10T18_50_35/path/to/file"
		if err != nil {
			return errors.Wrap(err, "failed to get relative path")
		}

		trimmedRelPath := SupportBundleNameRegex.ReplaceAllString(relPath, "")        // becomes: "path/to/file"
		trimmedRelPath = strings.TrimPrefix(trimmedRelPath, string(os.PathSeparator)) // extra measure to ensure no leading slashes. for example: "/path/to/file"
		if trimmedRelPath == "" {
			return nil
		}

		for _, filename := range filenames {
			trimmedFileName := SupportBundleNameRegex.ReplaceAllString(filename, "")
			trimmedFileName = strings.TrimPrefix(trimmedFileName, string(os.PathSeparator))
			if trimmedFileName == "" {
				continue
			}
			if trimmedRelPath == trimmedFileName {
				content, err := ioutil.ReadFile(path)
				if err != nil {
					return errors.Wrap(err, "failed to read file")
				}

				files[filename] = content
				return nil
			}
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk")
	}

	return files, nil
}

func GetSpecSecretName(appSlug string) string {
	return fmt.Sprintf("kotsadm-%s-supportbundle", appSlug)
}

func GetSpecURI(appSlug string) string {
	return fmt.Sprintf("secret/%s/%s", util.PodNamespace, GetSpecSecretName(appSlug))
}

func GetBundleCommand(appSlug string) []string {
	redactURIs := []string{redact.GetKotsadmRedactSpecURI(), redact.GetAppRedactSpecURI(appSlug)}
	redactors := strings.Join(redactURIs, ",")

	command := []string{
		"curl https://krew.sh/support-bundle | bash",
		fmt.Sprintf("kubectl support-bundle %s --redactors=%s\n", GetSpecURI(appSlug), redactors),
	}

	return command
}

// CreateSupportBundleDependencies generates k8s secrets and configmaps for the support bundle spec and redactors.
// These resources will be used when executing a support bundle collection
func CreateSupportBundleDependencies(appID string, sequence int64, opts types.TroubleshootOptions) (*types.SupportBundle, error) {
	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get app %s", appID)
	}

	archivePath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archivePath)

	err = store.GetStore().GetAppVersionArchive(a.ID, sequence, archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load current kotskinds")
	}

	supportBundle, err := CreateRenderedSpec(a.ID, sequence, kotsKinds, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rendered support bundle spec")
	}

	err = redact.GenerateKotsadmRedactSpec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to write kotsadm redact spec configmap")
	}
	redactURIs := []string{redact.GetKotsadmRedactSpecURI()}

	err = redact.CreateRenderedAppRedactSpec(a.ID, sequence, kotsKinds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write app redact spec configmap")
	}
	redactURIs = append(redactURIs, redact.GetAppRedactSpecURI(a.Slug))

	supportBundleObj := types.SupportBundle{
		AppID:      appID,
		URI:        GetSpecURI(a.Slug),
		RedactURIs: redactURIs,
		Progress: types.SupportBundleProgress{
			CollectorCount: len(supportBundle.Spec.Collectors),
		},
	}

	return &supportBundleObj, nil
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
			b, err := ioutil.ReadFile(path)
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

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		err = errors.Wrap(err, "failed to create temp dir")
		logger.Error(err)
		return err
	}
	defer os.RemoveAll(archiveDir)

	err = store.GetStore().GetAppVersionArchive(foundApp.ID, foundApp.CurrentSequence, archiveDir)
	if err != nil {
		err = errors.Wrap(err, "failed to get app version archive")
		logger.Error(err)
		return err
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
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

	analyzer.Spec.Analyzers = addDefaultAnalyzers(analyzer.Spec.Analyzers)
	analyzer.Spec.Analyzers = addDefaultDynamicAnalyzers(analyzer.Spec.Analyzers, foundApp)

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
