package prune

import (
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	supportbundletypes "github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/stretchr/testify/assert"
)

// fakeFileStore records the archive paths it was asked to delete.
type fakeFileStore struct {
	deleted []string
}

func (f *fakeFileStore) Init() error                                        { return nil }
func (f *fakeFileStore) WaitForReady(ctx context.Context) error             { return nil }
func (f *fakeFileStore) WriteArchive(path string, body io.ReadSeeker) error { return nil }
func (f *fakeFileStore) ReadArchive(path string) (string, error)            { return "", nil }
func (f *fakeFileStore) DeleteArchive(path string) error {
	f.deleted = append(f.deleted, path)
	return nil
}

func dv(parentSequence int64) *downstreamtypes.DownstreamVersion {
	return &downstreamtypes.DownstreamVersion{ParentSequence: parentSequence, Sequence: parentSequence}
}

func TestPruneAppVersions(t *testing.T) {
	appID := "app"
	cfg := config{appVersionCount: 2, deleteDelay: 0}

	t.Run("protects deployed and newer versions, keeps newest N of the rest", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		s := mock_store.NewMockStore(ctrl)
		fs := &fakeFileStore{}

		// deployed at 100; 105 is newer/pending; older prunable: 90,80,70,60,50
		dvs := &downstreamtypes.DownstreamVersions{
			CurrentVersion: dv(100),
			AllVersions:    []*downstreamtypes.DownstreamVersion{dv(105), dv(100), dv(90), dv(80), dv(70), dv(60), dv(50)},
		}
		s.EXPECT().FindDownstreamVersions(appID, false).Return(dvs, nil)

		// prunable older = [90,80,70,60,50]; keep newest 2 (90,80); delete 70,60,50
		for _, seq := range []int64{70, 60, 50} {
			s.EXPECT().DeleteAppVersion(appID, seq).Return(nil)
		}

		pruneAppVersions(s, fs, appID, cfg)

		assert.ElementsMatch(t, []string{"app/70.tar.gz", "app/60.tar.gz", "app/50.tar.gz"}, fs.deleted)
	})

	t.Run("no current version means no deletions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		s := mock_store.NewMockStore(ctrl)
		fs := &fakeFileStore{}

		s.EXPECT().FindDownstreamVersions(appID, false).Return(&downstreamtypes.DownstreamVersions{}, nil)
		// no DeleteAppVersion expectations -> gomock fails if any are called

		pruneAppVersions(s, fs, appID, cfg)

		assert.Empty(t, fs.deleted)
	})

	t.Run("prunable count within retention means no deletions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		s := mock_store.NewMockStore(ctrl)
		fs := &fakeFileStore{}

		dvs := &downstreamtypes.DownstreamVersions{
			CurrentVersion: dv(100),
			AllVersions:    []*downstreamtypes.DownstreamVersion{dv(100), dv(90), dv(80)},
		}
		s.EXPECT().FindDownstreamVersions(appID, false).Return(dvs, nil)

		pruneAppVersions(s, fs, appID, cfg)

		assert.Empty(t, fs.deleted)
	})
}

func TestPruneSupportBundles(t *testing.T) {
	appID := "app"
	cfg := config{supportBundleCount: 2, deleteDelay: 0}

	t.Run("deletes oldest beyond retention count", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		s := mock_store.NewMockStore(ctrl)

		// newest-first (as ListSupportBundles returns)
		bundles := []*supportbundletypes.SupportBundle{
			{ID: "b5", AppID: appID},
			{ID: "b4", AppID: appID},
			{ID: "b3", AppID: appID},
			{ID: "b2", AppID: appID},
			{ID: "b1", AppID: appID},
		}
		s.EXPECT().ListSupportBundles(appID).Return(bundles, nil)

		// keep newest 2 (b5,b4); delete b3,b2,b1
		for _, id := range []string{"b3", "b2", "b1"} {
			s.EXPECT().DeleteSupportBundle(id, appID).Return(nil)
		}

		pruneSupportBundles(s, appID, cfg)
	})

	t.Run("within retention count means no deletions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		s := mock_store.NewMockStore(ctrl)

		bundles := []*supportbundletypes.SupportBundle{
			{ID: "b2", AppID: appID},
			{ID: "b1", AppID: appID},
		}
		s.EXPECT().ListSupportBundles(appID).Return(bundles, nil)
		// no DeleteSupportBundle expectations -> gomock fails if any are called

		pruneSupportBundles(s, appID, cfg)
	})
}
