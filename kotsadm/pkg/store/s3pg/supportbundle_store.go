package s3pg

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	kotss3 "github.com/replicatedhq/kots/kotsadm/pkg/s3"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	supportbundletypes "github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	troubleshootredact "github.com/replicatedhq/troubleshoot/pkg/redact"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

func (s S3PGStore) ListSupportBundles(appID string) ([]*supportbundletypes.SupportBundle, error) {
	db := persistence.MustGetPGSession()
	// DANGER ZONE: changing sort order here affects what support bundle is shown in the analysis view.
	query := `select id, slug, watch_id, name, size, status, created_at, uploaded_at, is_archived from supportbundle where watch_id = $1 order by created_at desc`

	rows, err := db.Query(query, appID)
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

func (s S3PGStore) ListPendingSupportBundlesForApp(appID string) ([]*supportbundletypes.PendingSupportBundle, error) {
	db := persistence.MustGetPGSession()
	query := `select id, app_id, cluster_id from pending_supportbundle where app_id = $1`

	rows, err := db.Query(query, appID)
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

func (s S3PGStore) GetSupportBundleFromSlug(slug string) (*supportbundletypes.SupportBundle, error) {
	db := persistence.MustGetPGSession()
	query := `select id from supportbundle where slug = $1`
	row := db.QueryRow(query, slug)

	id := ""
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan id")
	}

	return s.GetSupportBundle(id)
}

func (s S3PGStore) GetSupportBundle(id string) (*supportbundletypes.SupportBundle, error) {
	db := persistence.MustGetPGSession()
	query := `select id, slug, watch_id, name, size, status, tree_index, created_at, uploaded_at, is_archived from supportbundle where slug = $1`
	row := db.QueryRow(query, id)

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

func (s S3PGStore) CreatePendingSupportBundle(id string, appID string, clusterID string) error {
	db := persistence.MustGetPGSession()
	query := `insert into pending_supportbundle (id, app_id, cluster_id, created_at) values ($1, $2, $3, $4)`

	_, err := db.Exec(query, id, appID, clusterID, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}

	return nil
}

func (s S3PGStore) CreateSupportBundle(id string, appID string, archivePath string, marshalledTree []byte) (*supportbundletypes.SupportBundle, error) {
	fi, err := os.Stat(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive")
	}

	// upload the bundle to s3
	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(filepath.Join("supportbundles", id, "supportbundle.tar.gz"))

	newSession := awssession.New(kotss3.GetConfig())

	s3Client := s3.New(newSession)

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open archive file")
	}

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Body:   f,
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to upload to s3")
	}

	db := persistence.MustGetPGSession()
	query := `insert into supportbundle (id, slug, watch_id, size, status, created_at, tree_index) values ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.Exec(query, id, id, appID, fi.Size(), "uploaded", time.Now(), marshalledTree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert support bundle")
	}

	return &types.SupportBundle{
		ID: id,
	}, nil
}

// GetSupportBundle will fetch the bundle archive and return a path to where it
// is stored. The caller is responsible for deleting.
func (s S3PGStore) GetSupportBundleArchive(bundleID string) (string, error) {
	logger.Debug("getting support bundle",
		zap.String("bundleID", bundleID))

	tmpDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	newSession := awssession.New(kotss3.GetConfig())

	bucket := aws.String(os.Getenv("S3_BUCKET_NAME"))
	key := aws.String(fmt.Sprintf("supportbundles/%s/supportbundle.tar.gz", bundleID))

	outputFile, err := os.Create(filepath.Join(tmpDir, "supportbundle.tar.gz"))
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}
	defer outputFile.Close()

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(outputFile,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		return "", errors.Wrapf(err, "failed to download support bundle archive %q from bucket %q", *key, *bucket)
	}

	return filepath.Join(tmpDir, "supportbundle.tar.gz"), nil
}

func (s S3PGStore) GetSupportBundleAnalysis(id string) (*supportbundletypes.SupportBundleAnalysis, error) {
	db := persistence.MustGetPGSession()
	query := `SELECT id, error, max_severity, insights, created_at FROM supportbundle_analysis where supportbundle_id = $1`
	row := db.QueryRow(query, id)

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

func (s S3PGStore) SetSupportBundleAnalysis(id string, insights []byte) error {
	db := persistence.MustGetPGSession()
	query := `update supportbundle set status = $1 where id = $2`

	_, err := db.Exec(query, "analyzed", id)
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}

	query = `insert into supportbundle_analysis (id, supportbundle_id, error, max_severity, insights, created_at) values ($1, $2, null, null, $3, $4)`
	_, err = db.Exec(query, ksuid.New().String(), id, insights, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to insert insights")
	}

	return nil
}

func (s S3PGStore) GetRedactions(bundleID string) (troubleshootredact.RedactionList, error) {
	db := persistence.MustGetPGSession()
	q := `select redact_report from supportbundle where id = $1`

	var redactString sql.NullString
	row := db.QueryRow(q, bundleID)
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

func (s S3PGStore) SetRedactions(bundleID string, redacts troubleshootredact.RedactionList) error {
	db := persistence.MustGetPGSession()

	redactBytes, err := json.Marshal(redacts)
	if err != nil {
		return errors.Wrap(err, "marshal redactionlist")
	}

	query := `update supportbundle set redact_report = $1 where id = $2`
	_, err = db.Exec(query, string(redactBytes), bundleID)
	if err != nil {
		return errors.Wrap(err, "failed to set support bundle redact report")
	}
	return nil
}

func (s S3PGStore) GetSupportBundleSpecForApp(id string) (string, error) {
	q := `select supportbundle_spec from app_version
	inner join app on app_version.app_id = app.id and app_version.sequence = app.current_sequence
	where app.id = $1`

	spec := ""

	db := persistence.MustGetPGSession()
	row := db.QueryRow(q, id)
	err := row.Scan(&spec)
	if err != nil {
		return "", err
	}
	return spec, nil
}
