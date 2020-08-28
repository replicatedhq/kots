package ocistore

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/kotsadm/pkg/app/types"
	downstreamtypes "github.com/replicatedhq/kots/kotsadm/pkg/downstream/types"
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
	DownstreamListConfigmapName = "kotsadm-downstreams"
	AppDownstreamsConfigMapName = "kotsadm-appdownstreams"
)

func (s OCIStore) AddAppToAllDownstreams(appID string) error {
	return ErrNotImplemented
}

func (s OCIStore) SetAppInstallState(appID string, state string) error {
	return ErrNotImplemented
}

func (s OCIStore) ListInstalledApps() ([]*apptypes.App, error) {
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

func (s OCIStore) GetAppIDFromSlug(slug string) (string, error) {
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

func (s OCIStore) GetApp(id string) (*apptypes.App, error) {
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

func (s OCIStore) GetAppFromSlug(slug string) (*apptypes.App, error) {
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

func (s OCIStore) CreateApp(name string, upstreamURI string, licenseData string, isAirgapEnabled bool) (*apptypes.App, error) {
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
			installState = "airgap_upload_pending"
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

	return s.GetApp(id)
}

func (s OCIStore) ListDownstreamsForApp(appID string) ([]downstreamtypes.Downstream, error) {
	appDownstreamsConfigMap, err := s.getConfigmap(AppDownstreamsConfigMapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app downstreams list configmap")
	}

	key := fmt.Sprintf("app:%s", appID)
	downstreamIDsMarshaled, ok := appDownstreamsConfigMap.Data[key]
	if !ok {
		return []downstreamtypes.Downstream{}, nil
	}
	downstreamIDs := []string{}
	if err := json.Unmarshal([]byte(downstreamIDsMarshaled), &downstreamIDs); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal downstream ids for app")
	}

	downstreamsConfigMap, err := s.getConfigmap(DownstreamListConfigmapName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get downsteams config map")
	}

	downstreams := []downstreamtypes.Downstream{}
	for _, downstreamData := range downstreamsConfigMap.Data {
		downstream := downstreamtypes.Downstream{}
		if err := json.Unmarshal([]byte(downstreamData), &downstream); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal app downstream data")
		}

		downstreams = append(downstreams, downstream)
	}

	return downstreams, nil
}

func (s OCIStore) ListAppsForDownstream(clusterID string) ([]*apptypes.App, error) {
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

func (c OCIStore) SetSnapshotSchedule(appID string, snapshotSchedule string) error {
	return ErrNotImplemented
}

func (c OCIStore) SetSnapshotTTL(appID string, snapshotTTL string) error {
	return ErrNotImplemented
}

func (s OCIStore) updateApp(app *apptypes.App) error {
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
