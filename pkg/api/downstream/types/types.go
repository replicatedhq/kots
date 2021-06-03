package types

import (
	"time"

	v1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
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
	VersionLabel             string                          `json:"versionLabel"`
	Status                   string                          `json:"status"`
	CreatedOn                *time.Time                      `json:"createdOn"`
	ParentSequence           int64                           `json:"parentSequence"`
	Sequence                 int64                           `json:"sequence"`
	ReleaseNotes             string                          `json:"releaseNotes"`
	DeployedAt               *time.Time                      `json:"deployedAt"`
	Source                   string                          `json:"source"`
	PreflightResult          string                          `json:"preflightResult,omitempty"`
	PreflightResultCreatedAt *time.Time                      `json:"preflightResultCreatedAt,omitempty"`
	PreflightSkipped         bool                            `json:"preflightSkipped"`
	DiffSummary              string                          `json:"diffSummary,omitempty"`
	DiffSummaryError         string                          `json:"diffSummaryError,omitempty"`
	CommitURL                string                          `json:"commitUrl,omitempty"`
	GitDeployable            bool                            `json:"gitDeployable,omitempty"`
	UpstreamReleasedAt       *time.Time                      `json:"upstreamReleasedAt,omitempty"`
	YamlErrors               []v1beta1.InstallationYAMLError `json:"yamlErrors,omitempty"`
}

type DownstreamOutput struct {
	DryrunStdout string `json:"dryrunStdout"`
	DryrunStderr string `json:"dryrunStderr"`
	ApplyStdout  string `json:"applyStdout"`
	ApplyStderr  string `json:"applyStderr"`
	RenderError  string `json:"renderError"`
}
