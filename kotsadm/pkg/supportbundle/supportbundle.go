package supportbundle

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	"github.com/segmentio/ksuid"
)

func List(appID string) ([]*types.SupportBundle, error) {
	db := persistence.MustGetPGSession()
	// DANGER ZONE: changing sort order here affects what support bundle is shown in the analysis view.
	query := `select id, slug, watch_id, name, size, status, created_at, uploaded_at, is_archived from supportbundle where watch_id = $1 order by created_at desc`

	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}

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

func GetFromSlug(slug string) (*types.SupportBundle, error) {
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

	return Get(id)
}

func Get(id string) (*types.SupportBundle, error) {
	db := persistence.MustGetPGSession()
	query := `select id, slug, watch_id, name, size, status, tree_index, created_at, uploaded_at, is_archived from supportbundle where slug = $1`
	row := db.QueryRow(query, id)

	var name sql.NullString
	var size sql.NullFloat64
	var treeIndex sql.NullString
	var uploadedAt sql.NullTime
	var isArchived sql.NullBool

	s := &types.SupportBundle{}
	if err := row.Scan(&s.ID, &s.Slug, &s.AppID, &name, &size, &s.Status, &treeIndex, &s.CreatedAt, &uploadedAt, &isArchived); err != nil {
		return nil, errors.Wrap(err, "failed to scan")
	}

	s.Name = name.String
	s.Size = size.Float64
	s.TreeIndex = treeIndex.String
	s.IsArchived = isArchived.Bool

	if uploadedAt.Valid {
		s.UploadedAt = &uploadedAt.Time
	}

	return s, nil
}

func Collect(appID string, clusterID string) error {
	db := persistence.MustGetPGSession()
	query := `insert into pending_supportbundle (id, app_id, cluster_id, created_at) values ($1, $2, $3, $4)`

	id := ksuid.New().String()

	_, err := db.Exec(query, id, appID, clusterID, time.Now())
	if err != nil {
		return errors.Wrap(err, "failed to insert support bundle")
	}

	return nil
}

func CreateBundle(requestedID string, appID string, archivePath string) (*types.SupportBundle, error) {
	fi, err := os.Stat(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read archive")
	}

	id := ksuid.New().String()
	if requestedID != "" {
		id = requestedID
	}

	storageBaseURI := os.Getenv("STORAGE_BASEURI")
	if storageBaseURI == "" {
		// KOTS 1.15 and earlier only supported s3 and there was no configuration
		storageBaseURI = fmt.Sprintf("s3://%s/%s", os.Getenv("S3_ENDPOINT"), os.Getenv("S3_BUCKET_NAME"))
	}

	parsedURI, err := url.Parse(storageBaseURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse storage base uri")
	}

	if parsedURI.Scheme == "docker" {
		if err := uploadBundleToDocker(id, archivePath, storageBaseURI); err != nil {
			return nil, errors.Wrap(err, "failed to upload to s3")
		}
	} else if parsedURI.Scheme == "s3" {
		if err := uploadBundleToS3(id, archivePath); err != nil {
			return nil, errors.Wrap(err, "failed to upload to s3")
		}
	}

	fileTree, err := archiveToFileTree(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate file tree")
	}

	marshaledTree, err := json.Marshal(fileTree.Nodes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal tree index")
	}

	db := persistence.MustGetPGSession()
	query := `insert into supportbundle (id, slug, watch_id, size, status, created_at, tree_index) values ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.Exec(query, id, id, appID, fi.Size(), "uploaded", time.Now(), marshaledTree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert support bundle")
	}

	return &types.SupportBundle{
		ID: id,
	}, nil
}

func GetFilesContents(bundleID string, filenames []string) (map[string][]byte, error) {
	bundleArchive, err := GetSupportBundle(bundleID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bundle")
	}
	defer os.RemoveAll(bundleArchive)

	tmpDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tmp dir")
	}
	defer os.RemoveAll(tmpDir)

	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(bundleArchive, tmpDir); err != nil {
		return nil, errors.Wrap(err, "failed to unarchive")
	}

	files := map[string][]byte{}
	for _, filename := range filenames {
		content, err := ioutil.ReadFile(filepath.Join(tmpDir, filename))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read  file")
		}

		files[filename] = content
	}

	return files, nil
}
