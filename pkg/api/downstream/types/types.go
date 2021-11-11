package types

import (
	"time"

	"github.com/blang/semver"
	v1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
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
	VersionLabel             string                             `json:"versionLabel"`
	Semver                   *semver.Version                    `json:"semver,omitempty"`
	Status                   storetypes.DownstreamVersionStatus `json:"status"`
	CreatedOn                *time.Time                         `json:"createdOn"`
	ParentSequence           int64                              `json:"parentSequence"`
	Sequence                 int64                              `json:"sequence"`
	ReleaseNotes             string                             `json:"releaseNotes"`
	DeployedAt               *time.Time                         `json:"deployedAt"`
	Source                   string                             `json:"source"`
	PreflightResult          string                             `json:"preflightResult,omitempty"`
	PreflightResultCreatedAt *time.Time                         `json:"preflightResultCreatedAt,omitempty"`
	PreflightSkipped         bool                               `json:"preflightSkipped"`
	DiffSummary              string                             `json:"diffSummary,omitempty"`
	DiffSummaryError         string                             `json:"diffSummaryError,omitempty"`
	CommitURL                string                             `json:"commitUrl,omitempty"`
	GitDeployable            bool                               `json:"gitDeployable,omitempty"`
	UpstreamReleasedAt       *time.Time                         `json:"upstreamReleasedAt,omitempty"`
	YamlErrors               []v1beta1.InstallationYAMLError    `json:"yamlErrors,omitempty"`
}

type DownstreamVersions struct {
	CurrentVersion  *DownstreamVersion
	PendingVersions []*DownstreamVersion
	PastVersions    []*DownstreamVersion
	AllVersions     []*DownstreamVersion
}

// Modified bubble sort: instead of comparing adjacent elements, compare the elements at the semvers only.
// Input is assumed to be sorded by sequence so non-semver elements are already in correct order.
func SortDownstreamVersions(versions *DownstreamVersions) {
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

type DownstreamOutput struct {
	DryrunStdout string `json:"dryrunStdout"`
	DryrunStderr string `json:"dryrunStderr"`
	ApplyStdout  string `json:"applyStdout"`
	ApplyStderr  string `json:"applyStderr"`
	HelmStdout   string `json:"helmStdout"`
	HelmStderr   string `json:"helmStderr"`
	RenderError  string `json:"renderError"`
}
