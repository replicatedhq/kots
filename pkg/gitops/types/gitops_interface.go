package types

type DownstreamGitOps interface {
	CreateGitOpsDownstreamCommit(appID string, clusterID string, newSequence float64, archiveDir string, downstreamName string) (string, error)
}
