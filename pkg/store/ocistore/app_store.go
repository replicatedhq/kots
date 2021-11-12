package ocistore

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/segmentio/ksuid"
)

/* AppStore
   The app store stores each version archive a single artifact in the registry
   The list of all apps is stored in a config map
   The list of all downstreams is stored in a config map
   The relation of apps->downstreams is stored in a config map
*/

const (
	AppListConfigmapName        = "kotsadm-apps"
	AppDownstreamsConfigMapName = "kotsadm-appdownstreams"
)

func (s *OCIStore) AddAppToAllDownstreams(appID string) error {
	clusters, err := s.ListClusters()
	if err != nil {
		return errors.Wrap(err, "failed to list clusters")
	}

	configMap, err := s.getConfigmap(AppDownstreamsConfigMapName)
	if err != nil {
		return errors.Wrap(err, "failed to get appdownstreams configmap")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	clusterIDs := []string{}
	for _, cluster := range clusters {
		clusterIDs = append(clusterIDs, cluster.ClusterID)
	}

	b, err := json.Marshal(clusterIDs)
	if err != nil {
		return errors.Wrap(err, "failed to marshal cluster ids")
	}

	configMap.Data[fmt.Sprintf("app.%s", appID)] = string(b)

	if err := s.updateConfigmap(configMap); err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func (s *OCIStore) SetAppInstallState(appID string, state string) error {
	app, err := s.GetApp(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	app.InstallState = state

	if err := s.updateApp(app); err != nil {
		return errors.Wrap(err, "failed to update app")
	}

	return nil
}

func (s *OCIStore) ListInstalledApps() ([]*apptypes.App, error) {
	appListConfigmap, err := s.getConfigmap(AppListConfigmapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app list configmap")
	}

	apps := []*apptypes.App{}
	for _, appData := range appListConfigmap.Data {
		app := apptypes.App{}
		if err := json.Unmarshal([]byte(appData), &app); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal app data")
		}

		apps = append(apps, &app)
	}

	return apps, nil
}

func (s *OCIStore) ListInstalledAppSlugs() ([]string, error) {
	apps, err := s.ListInstalledApps()
	if err != nil {
		return nil, err
	}
	appSlugs := []string{}
	for _, app := range apps {
		appSlugs = append(appSlugs, app.Slug)
	}
	return appSlugs, nil
}

func (s *OCIStore) GetAppIDFromSlug(slug string) (string, error) {
	appListConfigmap, err := s.getConfigmap(AppListConfigmapName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get app list configmap")
	}

	for _, appData := range appListConfigmap.Data {
		app := apptypes.App{}
		if err := json.Unmarshal([]byte(appData), &app); err != nil {
			return "", errors.Wrap(err, "failed to unmarshal app data")
		}

		if app.Slug == slug {
			return app.ID, nil
		}
	}

	return "", ErrNotFound
}

func (s *OCIStore) GetApp(id string) (*apptypes.App, error) {
	appListConfigmap, err := s.getConfigmap(AppListConfigmapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app list configmap")
	}

	app := apptypes.App{}
	if err := json.Unmarshal([]byte(appListConfigmap.Data[id]), &app); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal app data")
	}

	return &app, nil
}

func (s *OCIStore) GetAppFromSlug(slug string) (*apptypes.App, error) {
	appListConfigmap, err := s.getConfigmap(AppListConfigmapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app list configmap")
	}

	for _, appData := range appListConfigmap.Data {
		app := apptypes.App{}
		if err := json.Unmarshal([]byte(appData), &app); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal app data")
		}

		if app.Slug == slug {
			return &app, nil
		}
	}

	return nil, ErrNotFound
}

func (s *OCIStore) CreateApp(name string, upstreamURI string, licenseData string, isAirgapEnabled bool, skipImagePush bool, registryIsReadOnly bool) (*apptypes.App, error) {
	appListConfigmap, err := s.getConfigmap(AppListConfigmapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app list configmap")
	}

	existingAppSlugs := []string{}
	for _, appData := range appListConfigmap.Data {
		app := apptypes.App{}
		if err := json.Unmarshal([]byte(appData), &app); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal app data")
		}

		existingAppSlugs = append(existingAppSlugs, app.Slug)
	}

	titleForSlug := strings.Replace(name, ".", "-", 0)
	slugProposal := slug.Make(titleForSlug)

	foundUniqueSlug := false
	i := 0
	for !foundUniqueSlug {
		if i > 0 {
			slugProposal = fmt.Sprintf("%s-%d", titleForSlug, i)
		}

		foundUniqueSlug = true
		for _, existingAppSlug := range existingAppSlugs {
			if slugProposal == existingAppSlug {
				foundUniqueSlug = false
			}
		}
	}

	installState := ""
	if strings.HasPrefix(upstreamURI, "replicated://") == false {
		installState = "installed"
	} else {
		if isAirgapEnabled {
			if skipImagePush {
				installState = "installed"
			} else {
				installState = "airgap_upload_pending"
			}
		} else {
			installState = "online_upload_pending"
		}
	}

	id := ksuid.New().String()

	app := apptypes.App{
		ID:           id,
		Name:         name,
		IconURI:      "",
		CreatedAt:    time.Now(),
		Slug:         slugProposal,
		UpstreamURI:  upstreamURI,
		License:      licenseData,
		InstallState: installState,
	}
	b, err := json.Marshal(app)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal app")
	}

	if appListConfigmap.Data == nil {
		appListConfigmap.Data = map[string]string{}
	}

	appListConfigmap.Data[id] = string(b)
	if err := s.updateConfigmap(appListConfigmap); err != nil {
		return nil, errors.Wrap(err, "failed to update app list")
	}

	r, err := s.GetRegistryDetailsForApp(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app registry info")
	}
	err = s.UpdateRegistry(id, r.Hostname, r.Username, r.Password, r.Namespace, registryIsReadOnly)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update app registry info")
	}

	return s.GetApp(id)
}

func (s *OCIStore) ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error) {
	appDownstreamsConfigMap, err := s.getConfigmap(AppDownstreamsConfigMapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app downstreams list configmap")
	}

	key := fmt.Sprintf("app.%s", appID)
	downstreamIDsMarshaled, ok := appDownstreamsConfigMap.Data[key]
	if !ok {
		return []downstreamtypes.Downstream{}, nil
	}
	downstreamIDs := []string{}
	if err := json.Unmarshal([]byte(downstreamIDsMarshaled), &downstreamIDs); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal downstream ids for app")
	}

	clusters, err := s.ListClusters()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clusters")
	}

	matchingClusters := []downstreamtypes.Downstream{}
	for _, cluster := range clusters {
		for _, downstreamID := range downstreamIDs {
			if cluster.ClusterID == downstreamID {
				matchingClusters = append(matchingClusters, *cluster)
			}
		}
	}

	return matchingClusters, nil
}

func (s *OCIStore) ListAppsForDownstream(clusterID string) ([]*apptypes.App, error) {
	appDownstreamsConfigMap, err := s.getConfigmap(AppDownstreamsConfigMapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app downstreams list configmap")
	}

	key := fmt.Sprintf("downstream:%s", clusterID)
	appIDsMarshaled, ok := appDownstreamsConfigMap.Data[key]
	if !ok {
		return []*apptypes.App{}, nil
	}
	appIDs := []string{}
	if err := json.Unmarshal([]byte(appIDsMarshaled), &appIDs); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal downstream ids for app")
	}

	appsConfigmap, err := s.getConfigmap(AppListConfigmapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get downsteams config map")
	}

	apps := []*apptypes.App{}
	for _, appData := range appsConfigmap.Data {
		app := apptypes.App{}
		if err := json.Unmarshal([]byte(appData), &app); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal app data")
		}

		apps = append(apps, &app)
	}

	return apps, nil
}

func (c OCIStore) GetDownstream(clusterID string) (*downstreamtypes.Downstream, error) {
	return nil, ErrNotImplemented
}

func (c OCIStore) IsGitOpsEnabledForApp(appID string) (bool, error) {
	return false, ErrNotImplemented
}

func (c OCIStore) SetUpdateCheckerSpec(appID string, updateCheckerSpec string) error {
	return ErrNotImplemented
}

func (c OCIStore) SetSemverAutoDeploy(appID string, semverAutoDeploy apptypes.SemverAutoDeploy, semverAutoDeploySchedule string) error {
	return ErrNotImplemented
}

func (c OCIStore) SetSnapshotSchedule(appID string, snapshotSchedule string) error {
	return ErrNotImplemented
}

func (c OCIStore) SetSnapshotTTL(appID string, snapshotTTL string) error {
	return ErrNotImplemented
}

func (s *OCIStore) updateApp(app *apptypes.App) error {
	b, err := json.Marshal(app)
	if err != nil {
		return errors.Wrap(err, "failed to marhsal app")
	}

	configMap, err := s.getConfigmap(AppListConfigmapName)
	if err != nil {
		return errors.Wrap(err, "failed to get app list")
	}

	if configMap.Data == nil {
		configMap.Data = map[string]string{}
	}

	configMap.Data[app.ID] = string(b)

	if err := s.updateConfigmap(configMap); err != nil {
		return errors.Wrap(err, "failed to update app list config map")
	}

	return nil
}

func (s *OCIStore) RemoveApp(appID string) error {
	return ErrNotImplemented
}
