package ocistore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/ocidb/ocidb/pkg/ocidb"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	supportbundletypes "github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

func (s OCIStore) ListSupportBundles(appID string) ([]*supportbundletypes.SupportBundle, error) {
	// DANGER ZONE: changing sort order here affects what support bundle is shown in the analysis view.
	query := `select id, slug, watch_id, name, size, status, created_at, uploaded_at, is_archived from supportbundle where watch_id = $1 order by created_at desc`

	rows, err := s.connection.DB.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	supportBundles := []*types.SupportBundle{}

	for rows.Next() {
		var name sql.NullString
		var size sql.NullFloat64
		var uploadedAt sql.NullTime
		var isArchived sql.NullBool

		s := &types.SupportBundle{}
		if err := rows.Scan(&s.ID, &s.Slug, &s.AppID, &name, &size, &s.Status, &s.CreatedAt, &uploadedAt, &isArchived); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		s.Name = name.String
		s.Size = size.Float64
		s.IsArchived = isArchived.Bool

		if uploadedAt.Valid {
			s.UploadedAt = &uploadedAt.Time
		}

		supportBundles = append(supportBundles, s)
	}

	return supportBundles, nil
}

func (s OCIStore) ListPendingSupportBundlesForApp(appID string) ([]*supportbundletypes.PendingSupportBundle, error) {
	query := `select id, app_id, cluster_id from pending_supportbundle where app_id = $1`

	rows, err := s.connection.DB.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()

	pendingSupportBundles := []*supportbundletypes.PendingSupportBundle{}

	for rows.Next() {
		s := supportbundletypes.PendingSupportBundle{}
		if err := rows.Scan(&s.ID, &s.AppID, &s.ClusterID); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}

		pendingSupportBundles = append(pendingSupportBundles, &s)
	}

	return pendingSupportBundles, nil
}

func (s OCIStore) GetSupportBundleFromSlug(slug string) (*supportbundletypes.SupportBundle, error) {
	query := `select id from supportbundle where slug = $1`
	row := s.connection.DB.QueryRow(query, slug)

	id := ""
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan id")
	}

	return s.GetSupportBundle(id)
}

func (s OCIStore) GetSupportBundle(id string) (*supportbundletypes.SupportBundle, error) {
	query := `select id, slug, watch_id, name, size, status, tree_index, created_at, uploaded_at, is_archived from supportbundle where slug = $1`
	row := s.connection.DB.QueryRow(query, id)

	var name sql.NullString
	var size sql.NullFloat64
	var treeIndex sql.NullString
	var uploadedAt sql.NullTime
	var isArchived sql.NullBool

	supportbundle := &supportbundletypes.SupportBundle{}
	if err := row.Scan(&supportbundle.ID, &supportbundle.Slug, &supportbundle.AppID, &name, &size, &supportbundle.Status, &treeIndex, &supportbundle.CreatedAt, &uploadedAt, &isArchived); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	supportbundle.Name = name.String
	supportbundle.Size = size.Float64
	supportbundle.TreeIndex = treeIndex.String
	supportbundle.IsArchived = isArchived.Bool

	if uploadedAt.Valid {
		supportbundle.UploadedAt = &uploadedAt.Time
	}

	return supportbundle, nil
}

func (s OCIStore) CreatePendingSupportBundle(id string, appID string, clusterID string) error {
	query := `insert into pending_supportbundle (id, app_id, cluster_id, created_at) values ($1, $2, $3, $4)`

	_, err := s.connection.DB.Exec(query, id, appID, clusterID, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) CreateSupportBundle(id string, appID string, archivePath string, marshalledTree []byte) (*supportbundletypes.SupportBundle, error) {
	fi, err := os.Stat(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive")
	}

	fileContents, err := ioutil.ReadFile(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive file")
	}

	baseURI := os.Getenv("STORAGE_BASEURI")
	baseURI = strings.TrimSuffix(baseURI, "/")

	// docker images don't allow a large charset
	// so this names it registry.host/base/supportbundle:{bundle-id}
	ref := fmt.Sprintf("%s/supportbundle:%s", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(id))

	logger.Debug("pushing support bundle to docker registry",
		zap.String("ref", ref))

	options := docker.ResolverOptions{}

	registryHosts := func(host string) ([]docker.RegistryHost, error) {
		registryHost := docker.RegistryHost{
			Client:       http.DefaultClient,
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityPush,
		}

		if os.Getenv("STORAGE_BASEURI_PLAINHTTP") == "true" {
			registryHost.Scheme = "http"
		}

		return []docker.RegistryHost{
			registryHost,
		}, nil
	}

	options.Hosts = registryHosts

	resolver := docker.NewResolver(options)

	memoryStore := content.NewMemoryStore()
	desc := memoryStore.Add("supportbundle.tar.gz", "application/gzip", fileContents)
	pushContents := []ocispec.Descriptor{desc}
	pushedDescriptor, err := oras.Push(context.Background(), resolver, ref, memoryStore, pushContents)
	if err != nil {
		return nil, errors.Wrap(err, "failed to push archive to docker registry")
	}

	logger.Info("pushed support bundle to docker registry",
		zap.String("bundleID", id),
		zap.String("ref", ref),
		zap.String("digest", pushedDescriptor.Digest.String()))

	query := `insert into supportbundle (id, slug, watch_id, size, status, created_at, tree_index) values ($1, $2, $3, $4, $5, $6, $7)`

	_, err = s.connection.DB.Exec(query, id, id, appID, fi.Size(), "uploaded", time.Now(), marshalledTree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert support bundle")
	}

	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return nil, errors.Wrap(err, "failed to commit")
	}

	return &types.SupportBundle{
		ID: id,
	}, nil
}

// GetSupportBundle will fetch the bundle archive and return a path to where it
// is stored. The caller is responsible for deleting.
func (s OCIStore) GetSupportBundleArchive(bundleID string) (string, error) {
	logger.Debug("getting support bundle",
		zap.String("bundleID", bundleID))

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	fileStore := content.NewFileStore(tmpDir)
	defer fileStore.Close()

	allowedMediaTypes := []string{"application/gzip"}

	options := docker.ResolverOptions{}

	registryHosts := func(host string) ([]docker.RegistryHost, error) {
		registryHost := docker.RegistryHost{
			Client:       http.DefaultClient,
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityResolve | docker.HostCapabilityPull,
		}

		if os.Getenv("STORAGE_BASEURI_PLAINHTTP") == "true" {
			registryHost.Scheme = "http"
		}

		return []docker.RegistryHost{
			registryHost,
		}, nil
	}

	options.Hosts = registryHosts

	resolver := docker.NewResolver(options)

	baseURI := strings.TrimSuffix(os.Getenv("STORAGE_BASEURI"), "/")
	// docker images don't allow a large charset
	// so this names it registry.host/base/supportbundle:{bundle-id}
	ref := fmt.Sprintf("%s/supportbundle:%s", strings.TrimPrefix(baseURI, "docker://"), strings.ToLower(bundleID))

	pulledDescriptor, _, err := oras.Pull(context.Background(), resolver, ref, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return "", errors.Wrap(err, "failed to pull from registry storage")
	}

	logger.Debug("pulled support bundle from docker registry",
		zap.String("bundleID", bundleID),
		zap.String("ref", ref),
		zap.String("digest", pulledDescriptor.Digest.String()))

	return filepath.Join(tmpDir, "supportbundle.tar.gz"), nil
}

func (s OCIStore) GetSupportBundleAnalysis(id string) (*supportbundletypes.SupportBundleAnalysis, error) {
	query := `SELECT id, error, max_severity, insights, created_at FROM supportbundle_analysis where supportbundle_id = $1`
	row := s.connection.DB.QueryRow(query, id)

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

func (s OCIStore) SetSupportBundleAnalysis(id string, insights []byte) error {
	query := `update supportbundle set status = $1 where id = $2`

	_, err := s.connection.DB.Exec(query, "analyzed", id)
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}

	query = `insert into supportbundle_analysis (id, supportbundle_id, error, max_severity, insights, created_at) values ($1, $2, null, null, $3, $4)`
	_, err = s.connection.DB.Exec(query, ksuid.New().String(), id, insights, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to insert insights")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) GetRedactions(bundleID string) (troubleshootredact.RedactionList, error) {
	q := `select redact_report from supportbundle where id = $1`

	var redactString sql.NullString
	row := s.connection.DB.QueryRow(q, bundleID)
	err := row.Scan(&redactString)
	if err != nil {
		return troubleshootredact.RedactionList{}, errors.Wrap(err, "select redact_report")
	}

	if !redactString.Valid || redactString.String == "" {
		return troubleshootredact.RedactionList{}, fmt.Errorf("unable to find redactions for bundle %s", bundleID)
	}

	redacts := troubleshootredact.RedactionList{}
	err = json.Unmarshal([]byte(redactString.String), &redacts)
	if err != nil {
		return troubleshootredact.RedactionList{}, errors.Wrap(err, "unmarshal redact report")
	}

	return redacts, nil
}

func (s OCIStore) SetRedactions(bundleID string, redacts troubleshootredact.RedactionList) error {
	redactBytes, err := json.Marshal(redacts)
	if err != nil {
		return errors.Wrap(err, "marshal redactionlist")
	}

	query := `update supportbundle set redact_report = $1 where id = $2`
	_, err = s.connection.DB.Exec(query, string(redactBytes), bundleID)
	if err != nil {
		return errors.Wrap(err, "failed to set support bundle redact report")
	}
	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}

func (s OCIStore) GetSupportBundleSpecForApp(id string) (string, error) {
	q := `select supportbundle_spec from app_version
	inner join app on app_version.app_id = app.id and app_version.sequence = app.current_sequence
	where app.id = $1`

	spec := ""

	row := s.connection.DB.QueryRow(q, id)
	err := row.Scan(&spec)
	if err != nil {
		return "", err
	}
	return spec, nil
}
