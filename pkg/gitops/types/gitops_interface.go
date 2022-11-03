package types

type DownstreamGitOps interface {
	CreateGitOpsDownstreamCommit(appID string, clusterID string, newSequence int64, archiveDir string, downstreamName string) (string, error)
}
