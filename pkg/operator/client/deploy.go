package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/k8sutil"
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
	resources = sortResourcesForCreation(resources)

	// We don't dry run if there's a crd or a namespace because there's a likely chance that the
	// other docs have a custom resource using it
	shouldDryRun := !resources.HasCRDs() && !resources.HasNamespaces()
	if shouldDryRun {
		for _, resource := range resources {
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

	for _, resource := range resources {
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
	}

	return &deployRes, nil
}

func (c *Client) installWithHelm(helmDir string, kotsCharts []v1beta1.HelmChart) (*commandResult, error) {
	version := "3"
	chartsDir := filepath.Join(helmDir, "charts")

	orderedDirs, err := getSortedCharts(chartsDir, kotsCharts, c.TargetNamespace, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sorted helm charts")
	}

	var hasErr bool
	var multiStdout, multiStderr [][]byte
	for _, dir := range orderedDirs {
		installDir := filepath.Join(chartsDir, dir.Name)
		args := []string{"upgrade", "-i", dir.ReleaseName, installDir, "--timeout", "3600s"}

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
			multiStdout = append(multiStdout, []byte(fmt.Sprintf("------- %s -------", dir.ChartName)), stdout)
		}
		if len(stderr) > 0 {
			multiStderr = append(multiStderr, []byte(fmt.Sprintf("------- %s -------", dir.ChartName)), stderr)
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
}

func getSortedCharts(chartsDir string, kotsCharts []v1beta1.HelmChart, targetNamespace string, isUninstall bool) ([]orderedDir, error) {
	dirs, err := ioutil.ReadDir(chartsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive dir")
	}

	// get a list of the charts to be applied
	orderedDirs := []orderedDir{}
	for _, dir := range dirs {
		chartDir := filepath.Join(chartsDir, dir.Name())
		chartName, chartVersion, err := findChartNameAndVersion(chartDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find chart name and version in %s", chartDir)
		}
		orderedDirs = append(orderedDirs, orderedDir{
			Name:         dir.Name(),
			ChartName:    chartName,
			ChartVersion: chartVersion,
		})
	}

	// look through the list of kotsChart objects for each orderedDir, and if the name+version+dirname matches, use that weight+releasename
	// if there is no match, do not treat this as a fatal error
	for idx, dir := range orderedDirs {
		for _, kotsChart := range kotsCharts {
			if kotsChart.Spec.Chart.ChartVersion == dir.ChartVersion && kotsChart.Spec.Chart.Name == dir.ChartName && kotsChart.GetDirName() == dir.Name {
				orderedDirs[idx].Weight = kotsChart.Spec.Weight
				orderedDirs[idx].ReleaseName = kotsChart.GetReleaseName()
				orderedDirs[idx].Namespace = kotsChart.Spec.Namespace
				orderedDirs[idx].UpgradeFlags = kotsChart.Spec.HelmUpgradeFlags
			}
		}
		if orderedDirs[idx].ReleaseName == "" {
			// no matching kots chart was found, use the chart name as the release name
			orderedDirs[idx].ReleaseName = dir.ChartName
		}

		if orderedDirs[idx].Namespace == "" && targetNamespace != "" {
			orderedDirs[idx].Namespace = targetNamespace
		}
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
	chartFile, err := ioutil.ReadFile(chartfilePath)
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

func (c *Client) uninstallWithHelm(helmDir string, kotsCharts []v1beta1.HelmChart) error {
	version := "3"
	chartsDir := filepath.Join(helmDir, "charts")

	orderedDirs, err := getSortedCharts(chartsDir, kotsCharts, c.TargetNamespace, true)
	if err != nil {
		return errors.Wrap(err, "failed to get sorted helm charts")
	}

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

func getRemovedCharts(prevDir string, curDir string, previousKotsCharts []v1beta1.HelmChart, curKotsCharts []v1beta1.HelmChart) ([]v1beta1.HelmChart, error) {
	if prevDir == "" {
		return []v1beta1.HelmChart{}, nil
	}

	prevDirContent, err := ioutil.ReadDir(filepath.Join(prevDir, "charts"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list previous chart dir")
	}

	prevCharts := []v1beta1.HelmChart{}
	for _, f := range prevDirContent {
		if !f.IsDir() {
			continue
		}

		dirName := f.Name()
		prevChartDir := filepath.Join(prevDir, "charts", dirName)
		prevChartName, prevChartVersion, err := findChartNameAndVersion(prevChartDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find chart name and version in %s", prevChartDir)
		}

		for _, pkc := range previousKotsCharts {
			if pkc.Spec.Chart.ChartVersion == prevChartVersion && pkc.Spec.Chart.Name == prevChartName && pkc.GetDirName() == dirName {
				prevCharts = append(prevCharts, pkc)
			}
		}
	}

	if curDir == "" {
		return prevCharts, nil
	}

	curDirContent, err := ioutil.ReadDir(filepath.Join(curDir, "charts"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list current chart dir")
	}

	curCharts := []v1beta1.HelmChart{}
	for _, f := range curDirContent {
		if !f.IsDir() {
			continue
		}

		dirName := f.Name()
		curChartDir := filepath.Join(curDir, "charts", dirName)
		curChartName, curChartVersion, err := findChartNameAndVersion(curChartDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find chart name and version in %s", curChartDir)
		}

		for _, ckc := range curKotsCharts {
			if ckc.Spec.Chart.ChartVersion == curChartVersion && ckc.Spec.Chart.Name == curChartName && ckc.GetDirName() == dirName {
				curCharts = append(curCharts, ckc)
			}
		}
	}

	removedCharts := []v1beta1.HelmChart{}
	for _, prevChart := range prevCharts {
		found := false
		for _, curChart := range curCharts {
			if prevChart.Spec.Chart.ChartVersion != curChart.Spec.Chart.ChartVersion {
				continue
			}
			if prevChart.Spec.Chart.Name != curChart.Spec.Chart.Name {
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

func migrateExistingHelmReleaseSecrets(relaseName string, releaseNamespace string, kotsadmNamespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s client set")
	}
	return helm.MigrateExistingHelmReleaseSecrets(clientset, relaseName, releaseNamespace, kotsadmNamespace)
}
