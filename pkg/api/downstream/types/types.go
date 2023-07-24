package types

import (
	"sort"
	"time"

	"github.com/blang/semver"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotssemver "github.com/replicatedhq/kots/pkg/semver"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	v1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type Downstream struct {
	ClusterID        string `json:"id"`
	ClusterSlug      string `json:"slug"`
	Name             string `json:"name"`
	CurrentSequence  int64  `json:"currentSequence"`
	SnapshotSchedule string `json:"snapshotSchedule,omitempty"`
	SnapshotTTL      string `json:"snapshotTtl,omitempty"`
}

type DownstreamVersion struct {
	VersionLabel       string                             `json:"versionLabel"`
	Semver             *semver.Version                    `json:"semver,omitempty"`
	UpdateCursor       string                             `json:"updateCursor"`
	Cursor             *cursor.Cursor                     `json:"-"`
	ChannelID          string                             `json:"channelId,omitempty"`
	IsRequired         bool                               `json:"isRequired"`
	Status             storetypes.DownstreamVersionStatus `json:"status"`
	CreatedOn          *time.Time                         `json:"createdOn,omitempty"`
	ParentSequence     int64                              `json:"parentSequence"`
	Sequence           int64                              `json:"sequence"`
	DeployedAt         *time.Time                         `json:"deployedAt,omitempty"`
	Source             string                             `json:"source"`
	PreflightSkipped   bool                               `json:"preflightSkipped"`
	CommitURL          string                             `json:"commitUrl,omitempty"`
	GitDeployable      bool                               `json:"gitDeployable,omitempty"`
	UpstreamReleasedAt *time.Time                         `json:"upstreamReleasedAt,omitempty"`

	// The following fields are not queried by default and are only added as additional details when needed
	// because they make the queries really slow when there is a large number of versions
	IsDeployable               bool                            `json:"isDeployable,omitempty"`
	NonDeployableCause         string                          `json:"nonDeployableCause,omitempty"`
	ReleaseNotes               string                          `json:"releaseNotes,omitempty"`
	PreflightResult            string                          `json:"preflightResult,omitempty"`
	PreflightResultCreatedAt   *time.Time                      `json:"preflightResultCreatedAt,omitempty"`
	HasFailingStrictPreflights bool                            `json:"hasFailingStrictPreflights,omitempty"`
	HasConfig                  bool                            `json:"hasConfig,omitempty"`
	DiffSummary                string                          `json:"diffSummary,omitempty"`
	DiffSummaryError           string                          `json:"diffSummaryError,omitempty"`
	YamlErrors                 []v1beta1.InstallationYAMLError `json:"yamlErrors,omitempty"`
	NeedsKotsUpgrade           bool                            `json:"needsKotsUpgrade,omitempty"`
	KOTSKinds                  *kotsutil.KotsKinds             `json:"-"`
	DownloadStatus             DownloadStatus                  `json:"downloadStatus,omitempty"`
	AppTitle                   string                          `json:"appTitle,omitempty"`
	AppIconURI                 string                          `json:"appIconUri,omitempty"`
}

type DownloadStatus struct {
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

type DownstreamVersions struct {
	CurrentVersion  *DownstreamVersion
	PendingVersions []*DownstreamVersion
	PastVersions    []*DownstreamVersion
	AllVersions     []*DownstreamVersion
}

type DownstreamVersionHistory struct {
	VersionHistory         []*DownstreamVersion `json:"versionHistory"`
	TotalCount             int                  `json:"totalCount"`
	NumOfSkippedVersions   int                  `json:"numOfSkippedVersions"`
	NumOfRemainingVersions int                  `json:"numOfRemainingVersions"`
}

// SemverSortable interface implementations

type bySemver []*DownstreamVersion

func (v bySemver) Len() int {
	return len(v)
}

func (v bySemver) HasSemver(i int) bool {
	return v[i].Semver != nil
}

func (v bySemver) GetSemver(i int) *semver.Version {
	return v[i].Semver
}

func (v bySemver) GetSequence(i int) int64 {
	return v[i].Sequence
}

func (v bySemver) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// sort.Interface interface implementations

type bySequence []*DownstreamVersion

func (v bySequence) Len() int {
	return len(v)
}
func (v bySequence) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
func (v bySequence) Less(i, j int) bool {
	return v[i].Sequence < v[j].Sequence
}

// Modified bubble sort: instead of comparing adjacent elements, compare the elements at the semvers only.
// Input is assumed to be sorted by sequence so non-semver elements are already in correct order.
func SortDownstreamVersions(versions []*DownstreamVersion, sortBySemver bool) {
	if !sortBySemver {
		sort.Sort(sort.Reverse(bySequence(versions)))
		return
	}

	kotssemver.SortVersions(bySemver(versions))
}

// sort.Interface interface implementations

type byCursor []*DownstreamVersion

func (v byCursor) Len() int {
	return len(v)
}
func (v byCursor) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
func (v byCursor) Less(i, j int) bool {
	if v[i].Cursor == nil || v[j].Cursor == nil {
		return v[i].Sequence < v[j].Sequence
	}
	if (*v[i].Cursor).Equal(*v[j].Cursor) {
		return v[i].Sequence < v[j].Sequence
	}
	return (*v[i].Cursor).Before(*v[j].Cursor)
}

func SortDownstreamVersionsByCursor(allVersions []*DownstreamVersion) {
	sort.Sort(sort.Reverse(byCursor(allVersions)))
}

type DownstreamOutput struct {
	DryrunStdout string `json:"dryrunStdout"`
	DryrunStderr string `json:"dryrunStderr"`
	ApplyStdout  string `json:"applyStdout"`
	ApplyStderr  string `json:"applyStderr"`
	HelmStdout   string `json:"helmStdout"`
	HelmStderr   string `json:"helmStderr"`
	RenderError  string `json:"renderError"`
}
