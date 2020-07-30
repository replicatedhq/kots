package types

import (
	v1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"time"
)

type Downstream struct {
	ClusterID       string
	ClusterSlug     string
	Name            string
	CurrentSequence int64
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
	DiffSummary              string                          `json:"diffSummary,omitempty"`
	CommitURL                string                          `json:"commitUrl,omitempty"`
	GitDeployable            bool                            `json:"gitDeployable,omitempty"`
	YamlErrors               []v1beta1.InstallationYAMLError `json:"yamlErrors,omitempty"`
}
