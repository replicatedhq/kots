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
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/binaries"
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
	"k8s.io/apimachinery/pkg/labels"
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

func (c *Client) diffAndRemovePreviousManifests(deployArgs operatortypes.DeployAppArgs) error {
	decodedPrevious, err := base64.StdEncoding.DecodeString(deployArgs.PreviousManifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode previous manifests")
	}

	decodedCurrent, err := base64.StdEncoding.DecodeString(deployArgs.Manifests)
	if err != nil {
		return errors.Wrap(err, "failed to decode manifests")
	}

	targetNamespace := c.TargetNamespace
	if deployArgs.Namespace != "." {
		targetNamespace = deployArgs.Namespace
	}

	// we need to find the gvk+names that are present in the previous, but not in the current and then remove them
	// namespaces that were removed from YAML and added to additionalNamespaces should not be removed
	decodedPreviousStrings := strings.Split(string(decodedPrevious), "\n---\n")

	type previousObject struct {
		spec   string
		delete bool
	}
	decodedPreviousMap := map[string]previousObject{}
	for _, decodedPreviousString := range decodedPreviousStrings {
		k, o := GetGVKWithNameAndNs([]byte(decodedPreviousString), targetNamespace)

		delete := true
		if o.APIVersion == "v1" && o.Kind == "Namespace" {
			for _, n := range deployArgs.AdditionalNamespaces {
				if o.Metadata.Name == n {
					delete = false
					break
				}
			}
		}

		// if this is a restore process, only delete resources that are part of the backup and will be restored
		// e.g. resources that do not have the exclude label, and match the restore/backup label selector
		if deployArgs.IsRestore {
			if excludeLabel, exists := o.Metadata.Labels["velero.io/exclude-from-backup"]; exists && excludeLabel == "true" {
				delete = false
			}
			if deployArgs.RestoreLabelSelector != nil {
				s, err := metav1.LabelSelectorAsSelector(deployArgs.RestoreLabelSelector)
				if err != nil {
					return errors.Wrap(err, "failed to convert label selector to a selector")
				}
				if !s.Matches(labels.Set(o.Metadata.Labels)) {
					delete = false
				}
			}
		}

		decodedPreviousMap[k] = previousObject{
			spec:   decodedPreviousString,
			delete: delete,
		}
	}

	// now get the current names
	decodedCurrentStrings := strings.Split(string(decodedCurrent), "\n---\n")
	decodedCurrentMap := map[string]string{}
	for _, decodedCurrentString := range decodedCurrentStrings {
		k, _ := GetGVKWithNameAndNs([]byte(decodedCurrentString), targetNamespace)
		decodedCurrentMap[k] = decodedCurrentString
	}

	kubectl, err := binaries.GetKubectlPathForVersion(deployArgs.KubectlVersion)
	if err != nil {
		return errors.Wrap(err, "failed to find kubectl")
	}
	kustomize, err := binaries.GetKustomizePathForVersion(deployArgs.KustomizeVersion)
	if err != nil {
		return errors.Wrap(err, "failed to find kustomize")
	}
	config, err := k8sutil.GetClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	kubernetesApplier := applier.NewKubectl(kubectl, kustomize, config)

	// now remove anything that's in previous but not in current
	manifestsToDelete := []string{}
	for k, previous := range decodedPreviousMap {
		if _, ok := decodedCurrentMap[k]; ok {
			continue
		}
		if !previous.delete {
			continue
		}
		manifestsToDelete = append(manifestsToDelete, previous.spec)
	}

	deleteManifests(manifestsToDelete, targetNamespace, kubernetesApplier, DefaultDeletionPlan, deployArgs.Wait)

	if deployArgs.ClearPVCs {
		// TODO: multi-namespace support
		err := deletePVCs(targetNamespace, deployArgs.RestoreLabelSelector, deployArgs.AppSlug)
		if err != nil {
			return errors.Wrap(err, "failed to delete PVCs")
		}
	}

	if len(deployArgs.ClearNamespaces) > 0 {
		err := clearNamespaces(deployArgs.AppSlug, deployArgs.ClearNamespaces, deployArgs.IsRestore, deployArgs.RestoreLabelSelector, DefaultDeletionPlan)
		if err != nil {
			logger.Infof("Failed to clear namespaces: %v", err)
		}
	}

	return nil
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

	targetNamespace := c.TargetNamespace
	if deployArgs.Namespace != "." {
		targetNamespace = deployArgs.Namespace
	}

	kubernetesApplier, err := c.getApplier(deployArgs.KubectlVersion, deployArgs.KustomizeVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get applier")
	}

	decoded, err := base64.StdEncoding.DecodeString(deployArgs.Manifests)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode manifests")
	}

	firstApplyDocs, otherDocs, err := splitMutlidocYAMLIntoFirstApplyAndOthers(decoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split decoded into crds and other")
	}

	// We don't dry run if there's a crd because there's a likely chance that the
	// other docs has a custom resource using it
	shouldDryRun := firstApplyDocs == nil
	if shouldDryRun {

		byNamespace, err := docsByNamespace(decoded, targetNamespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get docs by requested namespace")
		}

		for requestedNamespace, docs := range byNamespace {
			if len(docs) == 0 {
				continue
			}

			logger.Infof("dry run applying manifests(s) in requested namespace: %s", requestedNamespace)
			dryrunStdout, dryrunStderr, dryRunErr := kubernetesApplier.ApplyCreateOrPatch(requestedNamespace, deployArgs.AppSlug, docs, true, deployArgs.Wait, deployArgs.AnnotateSlug)

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

			logger.Infof("dry run applied manifests(s) in requested namespace: %s", requestedNamespace)
		}

	}

	if len(firstApplyDocs) > 0 {
		logger.Info("applying first apply docs (CRDs, Namespaces)")

		// CRDs don't have namespaces, so we can skip splitting

		applyStdout, applyStderr, applyErr := kubernetesApplier.ApplyCreateOrPatch("", deployArgs.AppSlug, firstApplyDocs, false, deployArgs.Wait, deployArgs.AnnotateSlug)

		if len(applyStdout) > 0 {
			deployRes.applyResult.multiStdout = append(deployRes.applyResult.multiStdout, applyStdout)
		}
		if len(applyStderr) > 0 {
			deployRes.applyResult.multiStderr = append(deployRes.applyResult.multiStderr, applyStderr)
		}

		if applyErr != nil {
			logger.Infof("stdout (first apply) = %s", applyStdout)
			logger.Infof("stderr (first apply) = %s", applyStderr)
			logger.Infof("error (CRDS): %s", applyErr.Error())

			deployRes.applyResult.hasErr = true
			return &deployRes, nil
		}

		logger.Info("custom resource definition(s) applied")

		// Give the API server a minute (well, 5 seconds) to cache the CRDs
		time.Sleep(time.Second * 5)
	}

	byNamespace, err := docsByNamespace(otherDocs, targetNamespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get docs by requested namespace")
	}

	var hasErr bool
	var multiStdout, multiStderr [][]byte
	for requestedNamespace, docs := range byNamespace {
		if len(docs) == 0 {
			continue
		}

		logger.Infof("applying manifest(s) in namespace %s", requestedNamespace)
		applyStdout, applyStderr, applyErr := kubernetesApplier.ApplyCreateOrPatch(requestedNamespace, deployArgs.AppSlug, docs, false, deployArgs.Wait, deployArgs.AnnotateSlug)
		if applyErr != nil {
			logger.Infof("stdout (apply) = %s", applyStdout)
			logger.Infof("stderr (apply) = %s", applyStderr)
			logger.Infof("error: %s", applyErr.Error())
			hasErr = true
		} else {
			logger.Infof("manifest(s) applied in namespace %s", requestedNamespace)
		}
		if len(applyStdout) > 0 {
			multiStdout = append(multiStdout, applyStdout)
		}
		if len(applyStderr) > 0 {
			multiStderr = append(multiStderr, applyStderr)
		}
	}

	deployRes.applyResult.hasErr = hasErr
	deployRes.applyResult.multiStdout = multiStdout
	deployRes.applyResult.multiStderr = multiStderr

	return &deployRes, nil
}

func (c *Client) installWithHelm(helmDir string, targetNamespace string, kotsCharts []v1beta1.HelmChart) (*commandResult, error) {
	version := "3"
	chartsDir := filepath.Join(helmDir, "charts")

	orderedDirs, err := getSortedCharts(chartsDir, kotsCharts, targetNamespace)
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

func getSortedCharts(chartsDir string, kotsCharts []v1beta1.HelmChart, targetNamespace string) ([]orderedDir, error) {
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

		if orderedDirs[idx].Namespace == "" && targetNamespace != "" && targetNamespace != "." {
			orderedDirs[idx].Namespace = targetNamespace
		}
	}

	sort.Slice(orderedDirs, func(i, j int) bool {
		if orderedDirs[i].Weight != orderedDirs[j].Weight {
			return orderedDirs[i].Weight < orderedDirs[j].Weight
		}
		return orderedDirs[i].ChartName < orderedDirs[j].ChartName
	})
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

func (c *Client) uninstallWithHelm(helmDir string, targetNamespace string, charts []string) error {
	version := "3"

	for _, chart := range charts {
		args := []string{"uninstall", chart}
		if targetNamespace != "" && targetNamespace != "." {
			args = append(args, "-n", targetNamespace)
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
			return errors.Wrapf(err, "failed to uninstall chart %s: %s", chart, stderr)
		}
	}

	return nil
}

func getRemovedCharts(prevDir string, curDir string, previousKotsCharts []v1beta1.HelmChart) ([]string, error) {
	if prevDir == "" {
		return []string{}, nil
	}

	prevDirContent, err := ioutil.ReadDir(filepath.Join(prevDir, "charts"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list previous chart dir")
	}

	prevCharts := []string{}
	for _, f := range prevDirContent {
		if f.IsDir() {
			prevCharts = append(prevCharts, f.Name())
		}
	}

	if curDir == "" {
		return prevCharts, nil
	}

	curDirContent, err := ioutil.ReadDir(filepath.Join(curDir, "charts"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list current chart dir")
	}

	curCharts := []string{}
	for _, f := range curDirContent {
		if f.IsDir() {
			curCharts = append(curCharts, f.Name())
		}
	}

	removedCharts := []string{}
	for _, prevChart := range prevCharts {
		found := false
		for _, curChart := range curCharts {
			if prevChart == curChart {
				found = true
				break
			}
		}

		if !found {
			// find the release name that was used to install the chart
			prevChartDir := filepath.Join(prevDir, "charts", prevChart)
			prevChartName, prevChartVersion, err := findChartNameAndVersion(prevChartDir)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find chart name and version in %s", prevChartDir)
			}
			for _, prevKotsChart := range previousKotsCharts {
				if prevKotsChart.Spec.Chart.ChartVersion == prevChartVersion && prevKotsChart.Spec.Chart.Name == prevChartName && prevKotsChart.GetDirName() == prevChart {
					removedCharts = append(removedCharts, prevKotsChart.GetReleaseName())
				}
			}
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
