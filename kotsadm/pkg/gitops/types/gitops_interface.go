package types

type DownstreamGitOps interface {
	CreateGitOpsDownstreamCommit(appID string, clusterID string, newSequence int, archiveDir string, downstreamName string) (string, error)
}
