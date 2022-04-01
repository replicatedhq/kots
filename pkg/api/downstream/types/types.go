package types

import (
	"sort"
	"time"

	"github.com/blang/semver"
	v1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
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
	CreatedOn          *time.Time                         `json:"createdOn"`
	ParentSequence     int64                              `json:"parentSequence"`
	Sequence           int64                              `json:"sequence"`
	DeployedAt         *time.Time                         `json:"deployedAt"`
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
	DiffSummary                string                          `json:"diffSummary,omitempty"`
	DiffSummaryError           string                          `json:"diffSummaryError,omitempty"`
	YamlErrors                 []v1beta1.InstallationYAMLError `json:"yamlErrors,omitempty"`
	NeedsKotsUpgrade           bool                            `json:"needsKotsUpgrade,omitempty"`
	KOTSKinds                  *kotsutil.KotsKinds             `json:"-"`
	DownloadStatus             DownloadStatus                  `json:"downloadStatus,omitempty"`
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
func SortDownstreamVersions(versions *DownstreamVersions, bySemver bool) {
	if !bySemver {
		sort.Sort(sort.Reverse(bySequence(versions.AllVersions)))
		return
	}

	endIndex := len(versions.AllVersions)
	keepSorting := true
	for keepSorting {
		keepSorting = false
		for j := 0; j < endIndex-1; j++ {
			vj := versions.AllVersions[j]
			if vj.Semver == nil {
				continue
			}

			isLessThan := false
			for k := j + 1; k < endIndex; k++ {
				vk := versions.AllVersions[k]
				if vk.Semver == nil {
					continue
				}

				isLessThan = vj.Semver.LT(*vk.Semver)
				if vj.Semver.EQ(*vk.Semver) {
					isLessThan = vj.Sequence < vk.Sequence
				}

				if isLessThan {
					break
				}
			}

			if isLessThan {
				versions.AllVersions[j], versions.AllVersions[j+1] = versions.AllVersions[j+1], versions.AllVersions[j]
				keepSorting = true
			}
		}
	}
}

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
