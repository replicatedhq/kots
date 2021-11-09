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

type DownstreamVersions []DownstreamVersion

func (d DownstreamVersions) Len() int { return len(d) }

func (d DownstreamVersions) Less(i, j int) bool {
	if d[i].Semver == nil && d[j].Semver == nil {
		return d[i].Sequence < d[j].Sequence
	}
	if d[i].Semver == nil {
		return true
	}
	if d[j].Semver == nil {
		return false
	}
	return d[i].Semver.Compare(*d[j].Semver) == -1
}

func (d DownstreamVersions) Swap(i, j int) {
	tmp := d[i]
	d[i] = d[j]
	d[j] = tmp
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
