package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/appstate"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/applier"
	operatortypes "github.com/replicatedhq/kots/pkg/operator/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/yaml/v3"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var imagePullSecretsMtx sync.Mutex

type commandResult struct {
	hasErr      bool
	multiStdout [][]byte
	multiStderr [][]byte
}

type deployResult struct {
	dryRunResult commandResult
	applyResult  commandResult
}

func (c *Client) ensureNamespacePresent(name string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	_, err = clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		namespace := &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}

		_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create namespace")
		}
	}

	return nil
}

func (c *Client) ensureImagePullSecretsPresent(namespace string, imagePullSecrets []string) error {
	imagePullSecretsMtx.Lock()
	defer imagePullSecretsMtx.Unlock()

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	for _, secret := range imagePullSecrets {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(secret), nil, nil)
		if err != nil {
			return errors.Wrap(err, "failed to decode")
		}

		secret := obj.(*corev1.Secret)
		secret.Namespace = namespace

		foundSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
		if err != nil {
			if kuberneteserrors.IsNotFound(err) {
				// create it
				_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
				if err != nil {
					return errors.Wrap(err, "failed to create secret")
				}
			} else {
				return errors.Wrap(err, "failed to get secret")
			}
		} else {
			// Update it
			foundSecret.Data[".dockerconfigjson"] = secret.Data[".dockerconfigjson"]
			if _, err := clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
				return errors.Wrap(err, "failed to update secret")
			}
		}
	}

	return nil
}

func (c *Client) ensureResourcesPresent(deployArgs operatortypes.DeployAppArgs) (*deployResult, error) {
	var deployRes deployResult

	kubernetesApplier, err := c.getApplier(deployArgs.KubectlVersion, deployArgs.KustomizeVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get applier")
	}

	decoded, err := base64.StdEncoding.DecodeString(deployArgs.Manifests)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode manifests")
	}

	manifests := strings.Split(string(decoded), "\n---\n")
	resources := decodeManifests(manifests)
	phases := groupAndSortResourcesForCreation(resources)

	// We don't dry run if there's a crd or a namespace because there's a likely chance that the
	// other docs have a custom resource using it
	shouldDryRun := !resources.HasCRDs() && !resources.HasNamespaces()
	if shouldDryRun {
		for _, phase := range phases {
			logger.Infof("dry run applying phase %s", phase.Name)
			for _, resource := range phase.Resources {
				group := resource.GetGroup()
				version := resource.GetVersion()
				kind := resource.GetKind()
				name := resource.GetName()

				namespace := resource.GetNamespace()
				if namespace == "" {
					namespace = c.TargetNamespace
				}

				if resource.DecodeErrMsg == "" {
					logger.Infof("dry run applying resource %s/%s/%s/%s in namespace %s", group, version, kind, name, namespace)
				} else {
					logger.Infof("dry run applying unidentified resource. unable to parse error: %s", resource.DecodeErrMsg)
				}

				dryrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.ApplyCreateOrPatch(namespace, deployArgs.AppSlug, []byte(resource.Manifest), true, deployArgs.Wait, deployArgs.AnnotateSlug)
				if len(dryrunStdout) > 0 {
					deployRes.dryRunResult.multiStdout = append(deployRes.dryRunResult.multiStdout, dryrunStdout)
				}
				if len(dryrunStderr) > 0 {
					deployRes.dryRunResult.multiStderr = append(deployRes.dryRunResult.multiStderr, dryrunStderr)
				}

				if dryRunErr != nil {
					logger.Infof("stdout (dryrun) = %s", dryrunStdout)
					logger.Infof("stderr (dryrun) = %s", dryrunStderr)
					logger.Infof("error: %s", dryRunErr.Error())

					deployRes.dryRunResult.hasErr = true
					return &deployRes, nil
				}

				if resource.DecodeErrMsg == "" {
					logger.Infof("dry run applied resource %s/%s/%s/%s in namespace %s", group, version, kind, name, namespace)
				} else {
					logger.Info("dry run applied unidentified resource.")
				}
			}
		}
	}

	for _, phase := range phases {
		logger.Infof("applying phase %s", phase.Name)
		for _, resource := range phase.Resources {
			group := resource.GetGroup()
			version := resource.GetVersion()
			kind := resource.GetKind()
			name := resource.GetName()

			namespace := resource.GetNamespace()
			if namespace == "" {
				namespace = c.TargetNamespace
			}

			if resource.DecodeErrMsg == "" {
				logger.Infof("applying resource %s/%s/%s/%s in namespace %s", group, version, kind, name, namespace)
			} else {
				logger.Infof("applying unidentified resource. unable to parse error: %s", resource.DecodeErrMsg)
			}

			applyStdout, applyStderr, applyErr := kubernetesApplier.ApplyCreateOrPatch(namespace, deployArgs.AppSlug, []byte(resource.Manifest), false, deployArgs.Wait, deployArgs.AnnotateSlug)

			if len(applyStdout) > 0 {
				deployRes.applyResult.multiStdout = append(deployRes.applyResult.multiStdout, applyStdout)
			}
			if len(applyStderr) > 0 {
				deployRes.applyResult.multiStderr = append(deployRes.applyResult.multiStderr, applyStderr)
			}

			if applyErr != nil {
				logger.Infof("stdout (apply) = %s", applyStdout)
				logger.Infof("stderr (apply) = %s", applyStderr)
				logger.Infof("error: %s", applyErr.Error())

				deployRes.applyResult.hasErr = true
				return &deployRes, nil
			}

			if resource.DecodeErrMsg == "" {
				logger.Infof("applied resource %s/%s/%s/%s in namespace %s", group, version, kind, name, namespace)
			} else {
				logger.Info("applied unidentified resource.")
			}

			if resource.ShouldWaitForReady() {
				logger.Infof("waiting for resource %s/%s/%s/%s in namespace %s to be ready", group, version, kind, name, namespace)
				err := appstate.WaitForResourceToBeReady(namespace, name, resource.GVK)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to wait for resource %s/%s/%s/%s in namespace %s to be ready", group, version, kind, name, namespace)
				}
				logger.Infof("resource %s/%s/%s/%s in namespace %s is ready", group, version, kind, name, namespace)
			}

			if resource.ShouldWaitForProperties() {
				for _, prop := range resource.GetWaitForProperties() {
					logger.Infof("waiting for resource %s/%s/%s/%s in namespace %s to have property %s=%s", group, version, kind, name, namespace, prop.Path, prop.Value)
					err := appstate.WaitForProperty(namespace, name, resource.GVK, prop.Path, prop.Value)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to wait for resource %s/%s/%s/%s in namespace %s to have property %s=%s", group, version, kind, name, namespace, prop.Path, prop.Value)
					}
					logger.Infof("resource %s/%s/%s/%s in namespace %s has property %s=%s", group, version, kind, name, namespace, prop.Path, prop.Value)
				}
			}
		}
	}

	return &deployRes, nil
}

func (c *Client) installWithHelm(v1Beta1ChartsDir, v1beta2ChartsDir string, kotsCharts []kotsutil.HelmChartInterface) (*commandResult, error) {
	orderedDirs, err := getSortedCharts(v1Beta1ChartsDir, v1beta2ChartsDir, kotsCharts, c.TargetNamespace, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sorted charts")
	}

	version := "3"
	var hasErr bool
	var multiStdout, multiStderr [][]byte

	for _, dir := range orderedDirs {
		args := []string{"upgrade", "-i", dir.ReleaseName}
		if dir.APIVersion == "kots.io/v1beta1" {
			installDir := filepath.Join(v1Beta1ChartsDir, dir.Name)
			args = append(args, installDir)
		} else if dir.APIVersion == "kots.io/v1beta2" {
			installDir := filepath.Join(v1beta2ChartsDir, dir.Name)
			chartPath := filepath.Join(installDir, fmt.Sprintf("%s-%s.tgz", dir.ChartName, dir.ChartVersion))
			valuesPath := filepath.Join(installDir, "values.yaml")
			args = append(args, chartPath, "-f", valuesPath)
		} else {
			return nil, errors.Errorf("unknown api version %s", dir.APIVersion)
		}
		args = append(args, "--timeout", "3600s")

		if dir.Namespace != "" {
			args = append(args, "-n", dir.Namespace)

			// prior to kots v1.95.0, helm release secrets were created in the kotsadm namespace
			// Since kots v1.95.0, helm release secrets are created in the same namespace as the helm release
			// This migration will move the helm release secrets to the helm release namespace
			kotsadmNamespace := util.AppNamespace()
			if dir.Namespace != kotsadmNamespace {
				err := migrateExistingHelmReleaseSecrets(dir.ReleaseName, dir.Namespace, kotsadmNamespace)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to migrate helm release secrets for %s", dir.ReleaseName)
				}
			}
		}

		if len(dir.UpgradeFlags) > 0 {
			args = append(args, dir.UpgradeFlags...)
		}

		logger.Infof("running helm with arguments %v", args)
		cmd := exec.Command(fmt.Sprintf("helm%s", version), args...)
		stdout, stderr, err := applier.Run(cmd)
		if err != nil {
			logger.Infof("stdout (helm install) = %s", stdout)
			logger.Infof("stderr (helm install) = %s", stderr)
			logger.Infof("error: %s", err.Error())
			hasErr = true
		}

		if len(stdout) > 0 {
			multiStdout = append(multiStdout, []byte(fmt.Sprintf("------- %s -------", dir.Name)), stdout)
		}
		if len(stderr) > 0 {
			multiStderr = append(multiStderr, []byte(fmt.Sprintf("------- %s -------", dir.Name)), stderr)
		}
	}

	result := &commandResult{
		hasErr:      hasErr,
		multiStderr: multiStderr,
		multiStdout: multiStdout,
	}
	return result, nil
}

type orderedDir struct {
	Name         string
	Weight       int64
	ChartName    string
	ChartVersion string
	ReleaseName  string
	Namespace    string
	UpgradeFlags []string
	APIVersion   string
}

func getSortedCharts(v1Beta1ChartsDir string, v1Beta2ChartsDir string, kotsCharts []kotsutil.HelmChartInterface, targetNamespace string, isUninstall bool) ([]orderedDir, error) {
	// get a list of the chart directories
	foundDirs := []orderedDir{}

	if v1Beta1ChartsDir != "" {
		v1Beta1Dirs, err := os.ReadDir(v1Beta1ChartsDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read v1beta1 archive dir")
		}

		for _, dir := range v1Beta1Dirs {
			chartDir := filepath.Join(v1Beta1ChartsDir, dir.Name())
			chartName, chartVersion, err := findChartNameAndVersion(chartDir)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find chart name and version in %s", chartDir)
			}
			foundDirs = append(foundDirs, orderedDir{
				Name:         dir.Name(),
				ChartName:    chartName,
				ChartVersion: chartVersion,
				APIVersion:   "kots.io/v1beta1",
			})
		}
	}

	if v1Beta2ChartsDir != "" {
		v1Beta2Dirs, err := os.ReadDir(v1Beta2ChartsDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read v1beta2 archive dir")
		}

		for _, dir := range v1Beta2Dirs {
			chartDir := filepath.Join(v1Beta2ChartsDir, dir.Name())
			archivePath, err := findChartTgz(chartDir)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find chart tgz in %s", chartDir)
			}

			chartName, chartVersion, err := findChartNameAndVersionInArchive(archivePath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find chart name and version in %s", archivePath)
			}

			foundDirs = append(foundDirs, orderedDir{
				Name:         dir.Name(),
				ChartName:    chartName,
				ChartVersion: chartVersion,
				APIVersion:   "kots.io/v1beta2",
			})
		}
	}

	// look through the list of kotsChart objects and find the matching directory
	orderedDirs := []orderedDir{}
	for _, kotsChart := range kotsCharts {
		for idx, dir := range foundDirs {
			if kotsChart.GetDirName() != dir.Name {
				continue
			}
			if kotsChart.GetChartName() != dir.ChartName {
				continue
			}
			if kotsChart.GetChartVersion() != dir.ChartVersion {
				continue
			}
			if kotsChart.GetAPIVersion() != dir.APIVersion {
				continue
			}

			foundDirs[idx].Weight = kotsChart.GetWeight()
			foundDirs[idx].ReleaseName = kotsChart.GetReleaseName()
			foundDirs[idx].UpgradeFlags = kotsChart.GetUpgradeFlags()
			foundDirs[idx].Namespace = kotsChart.GetNamespace()
			if foundDirs[idx].Namespace == "" && targetNamespace != "" {
				foundDirs[idx].Namespace = targetNamespace
			}

			orderedDirs = append(orderedDirs, foundDirs[idx])

			// remove from foundDirs
			if idx == len(foundDirs)-1 {
				foundDirs = foundDirs[:idx]
			} else {
				foundDirs = append(foundDirs[:idx], foundDirs[idx+1:]...)
			}
			break
		}
	}

	// log any chart dirs that do not have a matching kotsChart
	for _, dir := range foundDirs {
		logger.Warnf("%s chart %s-%s in dir %s was not found in the kotskinds", dir.APIVersion, dir.ChartName, dir.ChartVersion, dir.Name)
	}

	if isUninstall {
		// higher weight should be uninstalled first
		sort.Slice(orderedDirs, func(i, j int) bool {
			if orderedDirs[i].Weight != orderedDirs[j].Weight {
				return orderedDirs[i].Weight > orderedDirs[j].Weight
			}
			return orderedDirs[i].ChartName > orderedDirs[j].ChartName
		})
	} else {
		// lower weight should be installed first
		sort.Slice(orderedDirs, func(i, j int) bool {
			if orderedDirs[i].Weight != orderedDirs[j].Weight {
				return orderedDirs[i].Weight < orderedDirs[j].Weight
			}
			return orderedDirs[i].ChartName < orderedDirs[j].ChartName
		})
	}

	return orderedDirs, nil
}

func findChartNameAndVersion(chartDir string) (string, string, error) {
	chartfilePath := filepath.Join(chartDir, "Chart.yaml")
	chartFile, err := os.ReadFile(chartfilePath)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to parse %s", chartfilePath)
	}
	chartInfo := struct {
		ChartName    string `yaml:"name"`
		ChartVersion string `yaml:"version"`
	}{}
	if err := yaml.Unmarshal(chartFile, &chartInfo); err != nil {
		return "", "", errors.Wrapf(err, "failed to unmarshal %s", chartfilePath)
	}
	return chartInfo.ChartName, chartInfo.ChartVersion, nil
}

func findChartTgz(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read dir %s", dir)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		path := filepath.Join(dir, file.Name())
		bytes, err := os.ReadFile(path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read file %s", path)
		}
		if archives.IsTGZ(bytes) {
			return path, nil
		}
	}

	return "", errors.New("no tgz found")
}

func findChartNameAndVersionInArchive(archivePath string) (string, string, error) {
	tmpDir, err := os.MkdirTemp("", "kots")
	if err != nil {
		return "", "", errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tmpDir)

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
			StripComponents:        1, // remove the top level folder
		},
	}
	if err := tarGz.Unarchive(archivePath, tmpDir); err != nil {
		return "", "", errors.Wrap(err, "failed to unarchive")
	}

	return findChartNameAndVersion(tmpDir)
}

func (c *Client) uninstallWithHelm(v1Beta1ChartsDir, v1Beta2ChartsDir string, kotsCharts []kotsutil.HelmChartInterface) error {
	orderedDirs, err := getSortedCharts(v1Beta1ChartsDir, v1Beta2ChartsDir, kotsCharts, c.TargetNamespace, true)
	if err != nil {
		return errors.Wrap(err, "failed to get sorted charts")
	}

	version := "3"

	for _, dir := range orderedDirs {
		args := []string{"uninstall", dir.ReleaseName}

		if dir.Namespace != "" {
			args = append(args, "-n", dir.Namespace)
		}

		logger.Infof("running helm with arguments %v", args)
		cmd := exec.Command(fmt.Sprintf("helm%s", version), args...)
		stdout, stderr, err := applier.Run(cmd)
		logger.Infof("stdout (helm uninstall) = %s", stdout)
		logger.Infof("stderr (helm uninstall) = %s", stderr)
		if err != nil {
			if strings.Contains(string(stderr), "not found") {
				continue
			}
			logger.Errorf("error: %s", err.Error())
			return errors.Wrapf(err, "failed to uninstall release %s for chart %s: %s", dir.ReleaseName, dir.ChartName, stderr)
		}
	}

	return nil
}

type getRemovedChartsOptions struct {
	prevV1Beta1Dir            string
	curV1Beta1Dir             string
	previousV1Beta1KotsCharts []kotsutil.HelmChartInterface
	currentV1Beta1KotsCharts  []kotsutil.HelmChartInterface
	prevV1Beta2Dir            string
	curV1Beta2Dir             string
	previousV1Beta2KotsCharts []kotsutil.HelmChartInterface
	currentV1Beta2KotsCharts  []kotsutil.HelmChartInterface
}

// getRemovedCharts returns a list of helm release names that were removed in the current version
func getRemovedCharts(opts getRemovedChartsOptions) ([]kotsutil.HelmChartInterface, error) {
	prevCharts := []kotsutil.HelmChartInterface{}

	if opts.prevV1Beta1Dir != "" {
		prevV1Beta1ChartsDir := filepath.Join(opts.prevV1Beta1Dir, "charts")
		matching, err := findMatchingHelmCharts(prevV1Beta1ChartsDir, opts.previousV1Beta1KotsCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find previous matching v1beta1 charts")
		}
		prevCharts = append(prevCharts, matching...)
	}

	if opts.prevV1Beta2Dir != "" {
		prevV1Beta2ChartsDir := filepath.Join(opts.prevV1Beta2Dir, "helm")
		matching, err := findMatchingHelmCharts(prevV1Beta2ChartsDir, opts.previousV1Beta2KotsCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find previous matching v1beta2 charts")
		}
		prevCharts = append(prevCharts, matching...)
	}

	curCharts := []kotsutil.HelmChartInterface{}

	if opts.curV1Beta1Dir != "" {
		curV1Beta1ChartsDir := filepath.Join(opts.curV1Beta1Dir, "charts")
		matching, err := findMatchingHelmCharts(curV1Beta1ChartsDir, opts.currentV1Beta1KotsCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find current matching v1beta1 charts")
		}
		curCharts = append(curCharts, matching...)
	}

	if opts.curV1Beta2Dir != "" {
		curV1Beta2ChartsDir := filepath.Join(opts.curV1Beta2Dir, "helm")
		matching, err := findMatchingHelmCharts(curV1Beta2ChartsDir, opts.currentV1Beta2KotsCharts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find current matching v1beta2 charts")
		}
		curCharts = append(curCharts, matching...)
	}

	removedCharts := []kotsutil.HelmChartInterface{}
	for _, prevChart := range prevCharts {
		found := false
		for _, curChart := range curCharts {
			if prevChart.GetNamespace() != curChart.GetNamespace() {
				continue
			}
			if prevChart.GetDirName() != curChart.GetDirName() {
				continue
			}
			found = true
			break
		}

		if !found {
			removedCharts = append(removedCharts, prevChart)
		}
	}

	return removedCharts, nil
}

func findMatchingHelmCharts(chartsDir string, kotsCharts []kotsutil.HelmChartInterface) ([]kotsutil.HelmChartInterface, error) {
	dirContent, err := os.ReadDir(chartsDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list chart dir %s", chartsDir)
	}

	matching := []kotsutil.HelmChartInterface{}

	for _, kotsChart := range kotsCharts {
		for _, f := range dirContent {
			if !f.IsDir() {
				continue
			}

			dirName := f.Name()
			chartDir := filepath.Join(chartsDir, dirName)

			var chartName, chartVersion string
			if kotsChart.GetAPIVersion() == "kots.io/v1beta1" {
				// v1beta1 charts are already unpacked, so we can just read the metadata
				chartName, chartVersion, err = findChartNameAndVersion(chartDir)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to find chart name and version in %s", chartDir)
				}
			} else if kotsChart.GetAPIVersion() == "kots.io/v1beta2" {
				// v1beta2 charts are packaged as tgz, so we need to find and extract it to get the chart name and version
				archivePath, err := findChartTgz(chartDir)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to find chart tgz in %s", chartDir)
				}
				chartName, chartVersion, err = findChartNameAndVersionInArchive(archivePath)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to find chart name and version in %s", chartDir)
				}
			} else {
				return nil, errors.Errorf("unknown api version %s", kotsChart.GetAPIVersion())
			}

			if kotsChart.GetChartVersion() == chartVersion && kotsChart.GetChartName() == chartName && kotsChart.GetDirName() == dirName {
				matching = append(matching, kotsChart)
				break
			}
		}
	}

	return matching, nil
}

func migrateExistingHelmReleaseSecrets(relaseName string, releaseNamespace string, kotsadmNamespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}
	return helm.MigrateExistingHelmReleaseSecrets(clientset, relaseName, releaseNamespace, kotsadmNamespace)
}
