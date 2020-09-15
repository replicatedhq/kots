package supportbundle

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	"github.com/segmentio/ksuid"
)

// Collect will queue collection of a new support bundle
func Collect(appID string, clusterID string) error {
	id := ksuid.New().String()

	return store.GetStore().CreatePendingSupportBundle(id, appID, clusterID)
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

func ClearPending(id string) error {
	db := persistence.MustGetPGSession()
	query := `delete from pending_supportbundle where id = $1`

	_, err := db.Exec(query, id)
	if err != nil {
		return errors.Wrap(err, "failed to exec")
	}

	return nil
}
