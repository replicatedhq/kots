package version

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/api/version/types"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	applicationv1beta1 "sigs.k8s.io/application/api/v1beta1"
)

type DownstreamGitOps struct {
}

func (d *DownstreamGitOps) CreateGitOpsDownstreamCommit(appID string, clusterID string, newSequence int, filesInDir string, downstreamName string) (string, error) {
	downstreamGitOps, err := gitops.GetDownstreamGitOps(appID, clusterID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get downstream gitops")
	}
	if downstreamGitOps == nil {
		return "", nil
	}

	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app")
	}
	createdCommitURL, err := gitops.CreateGitOpsCommit(downstreamGitOps, a.Slug, a.Name, int(newSequence), filesInDir, downstreamName)
	if err != nil {
		return "", errors.Wrap(err, "failed to create gitops commit")
	}

	return createdCommitURL, nil
}

// DeployVersion deploys the version for the given sequence
func DeployVersion(appID string, sequence int64) error {
	db := persistence.MustGetDBSession()

	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin")
	}
	defer tx.Rollback()

	query := `update app_downstream set current_sequence = $1 where app_id = $2`
	_, err = tx.Exec(query, sequence, appID)
	if err != nil {
		return errors.Wrap(err, "failed to update app downstream current sequence")
	}

	query = `update app_downstream_version set status = 'deployed', applied_at = $3 where sequence = $1 and app_id = $2`
	_, err = tx.Exec(query, sequence, appID, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to update app downstream version status")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func GetRealizedLinksFromAppSpec(appID string, sequence int64) ([]types.RealizedLink, error) {
	db := persistence.MustGetDBSession()
	query := `select app_spec, kots_app_spec from app_version where app_id = $1 and sequence = $2`
	row := db.QueryRow(query, appID, sequence)

	var appSpecStr sql.NullString
	var kotsAppSpecStr sql.NullString
	if err := row.Scan(&appSpecStr, &kotsAppSpecStr); err != nil {
		if err == sql.ErrNoRows {
			return []types.RealizedLink{}, nil
		}
		return nil, errors.Wrap(err, "failed to scan")
	}

	if appSpecStr.String == "" {
		return []types.RealizedLink{}, nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(appSpecStr.String), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode app spec yaml")
	}
	appSpec := obj.(*applicationv1beta1.Application)

	obj, _, err = decode([]byte(kotsAppSpecStr.String), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode kots app spec yaml")
	}
	kotsAppSpec := obj.(*kotsv1beta1.Application)

	realizedLinks := []types.RealizedLink{}
	for _, link := range appSpec.Spec.Descriptor.Links {
		rewrittenURL := link.URL
		for _, port := range kotsAppSpec.Spec.ApplicationPorts {
			if port.ApplicationURL == link.URL {
				rewrittenURL = fmt.Sprintf("http://localhost:%d", port.LocalPort)
			}
		}
		realizedLink := types.RealizedLink{
			Title: link.Description,
			Uri:   rewrittenURL,
		}
		realizedLinks = append(realizedLinks, realizedLink)
	}

	return realizedLinks, nil
}

func GetForwardedPortsFromAppSpec(appID string, sequence int64) ([]types.ForwardedPort, error) {
	appVersion, err := store.GetStore().GetAppVersion(appID, sequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version")
	}

	if appVersion.KOTSKinds == nil {
		return nil, errors.Wrap(err, "failed to get kots kinds for app")
	}

	kotsAppSpec := appVersion.KOTSKinds.KotsApplication

	if len(kotsAppSpec.Spec.ApplicationPorts) == 0 {
		return []types.ForwardedPort{}, nil
	}

	ports := []types.ForwardedPort{}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	appNamespace := util.AppNamespace()

	// To forward the ports, we need to have the port listed
	// in the kots spec only
	for _, port := range kotsAppSpec.Spec.ApplicationPorts {
		// make a best effort to not return ports to services that are not yet ready
		// this is best effort because the service could restart at any time
		// and the RBAC persona that this api is running as does not match
		// the users RBAC persona. Finally, this will not work in a gitops-application
		// unless it's deployed to the same namespace as the admin console
		// This has always been a limitation though, we need to design for this

		svc, err := clientset.CoreV1().Services(appNamespace).Get(context.TODO(), port.ServiceName, metav1.GetOptions{})
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get service to check status, namespace = %s", util.PodNamespace))
			continue
		}

		options := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String()}
		podList, err := clientset.CoreV1().Pods(appNamespace).List(context.TODO(), options)
		if err != nil {
			logger.Error(errors.Wrap(err, "failed to list pods in service"))
			continue
		}

		hasReadyPod := false
		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodRunning {
				for _, status := range pod.Status.ContainerStatuses {
					if status.Ready {
						hasReadyPod = true
					}
				}
			}
		}

		if !hasReadyPod {
			logger.Info("not forwarding to service because no pods are ready", zap.String("serviceName", port.ServiceName), zap.String("namespace", util.PodNamespace))
			continue
		}

		ports = append(ports, types.ForwardedPort{
			ServiceName:    port.ServiceName,
			ServicePort:    port.ServicePort,
			LocalPort:      port.LocalPort,
			ApplicationURL: port.ApplicationURL,
		})
	}

	return ports, nil
}

// GetBaseArchiveDirForVersion returns the base archive directory for a given version label.
// the base archive directory contains data such as config values.
// caller is responsible for cleaning up the created archive dir.
func GetBaseArchiveDirForVersion(appID string, clusterID string, targetVersionLabel string) (string, error) {
	appVersions, err := store.GetStore().GetAppVersions(appID, clusterID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get app versions for app %s", appID)
	}
	if len(appVersions.AllVersions) == 0 {
		return "", errors.Errorf("no app versions found for app %s in downstream %s", appID, clusterID)
	}

	mockVersion := &downstreamtypes.DownstreamVersion{
		// to id the mocked version and be able to retrieve it later.
		// use "MaxInt64" so that it ends up on the top of the list if it's not a semvered version.
		Sequence: math.MaxInt64,
	}

	targetSemver, err := semver.ParseTolerant(targetVersionLabel)
	if err == nil {
		mockVersion.Semver = &targetSemver
	}

	// add to the top of the list and sort
	appVersions.AllVersions = append([]*downstreamtypes.DownstreamVersion{mockVersion}, appVersions.AllVersions...)
	downstreamtypes.SortDownstreamVersions(appVersions)

	var baseVersion *downstreamtypes.DownstreamVersion
	for i, v := range appVersions.AllVersions {
		if v.Sequence == math.MaxInt64 {
			// this is our mocked version, base it off of the previous version in the sorted list (if exists).
			if i < len(appVersions.AllVersions)-1 {
				baseVersion = appVersions.AllVersions[i+1]
			}
			// remove the mocked version from the list to not affect what the latest version is in case there's no previous version to use as base.
			appVersions.AllVersions = append(appVersions.AllVersions[:i], appVersions.AllVersions[i+1:]...)
			break
		}
	}

	// if a previous version was not found, base off of the latest version
	if baseVersion == nil {
		baseVersion = appVersions.AllVersions[0]
	}

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	err = store.GetStore().GetAppVersionArchive(appID, baseVersion.ParentSequence, archiveDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app version archive")
	}

	return archiveDir, nil
}
