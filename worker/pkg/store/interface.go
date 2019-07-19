package store

import (
	"context"
	"mime/multipart"

	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

type Store interface {
	GetInit(ctx context.Context, initID string) (*types.InitSession, error)
	SetInitStatus(ctx context.Context, initID string, status string) error
	GetUnfork(ctx context.Context, unforkID string) (*types.UnforkSession, error)
	SetUnforkStatus(ctx context.Context, unforkID string, status string) error
	CreateWatchFromState(ctx context.Context, stateJSON []byte, metadata []byte, title string, iconURI string, slug string, userID string, initID string, clusterID string, githubPath string, parentWatchID string) error

	GetSupportBundleURL(supportBundle *types.SupportBundle) (string, error)
	GetS3StoreURL(shipSession types.Session) (string, error)
	SetOutputFilepath(ctx context.Context, session types.Output) error
	UploadToS3(ctx context.Context, outputSession types.Output, file multipart.File) error
	DownloadFromS3(ctx context.Context, path string) (string, error)

	GetNextUploadSequence(ctx context.Context, watchID string) (int, error)
	UpdateWatchState(ctx context.Context, watchID string, stateJSON []byte, metadata []byte) error

	ListReadyUpdateIDs(ctx context.Context) ([]string, error)
	GetUpdate(ctx context.Context, updateID string) (*types.UpdateSession, error)
	SetUpdateStatus(ctx context.Context, updateID string, status string) error
	SetUpdateStarted(ctx context.Context, updateID string) error
	SetUpdateLogs(ctx context.Context, updateID string, podLogs string) error
	GetEdit(ctx context.Context, editID string) (*types.EditSession, error)
	SetEditStatus(ctx context.Context, edit string, status string) error

	GetParentWatchID(ctx context.Context, watchID string) (*string, error)
	ListReadyWatchIDs(ctx context.Context) ([]string, error)
	GetWatchIDFromSlug(ctx context.Context, slug string, userID string) (string, error)
	GetWatch(ctx context.Context, watchID string) (*types.Watch, error)
	GetWatches(ctx context.Context, userID string) ([]*types.Watch, error)
	CreateWatchVersion(ctx context.Context, watchID string, versionLabel string, status string, sourceBranch string, sequence int, pullRequestNumner int, setCurrent bool, parentSequence *int) error
	GetMostRecentWatchVersion(ctx context.Context, watchID string) (*types.WatchVersion, error)
	GetOneWatchVersion(ctx context.Context, watchID string, sequence int) (*types.WatchVersion, error)
	CreateWatchUpdate(ctx context.Context, watchID string, parentSequence *int) error
	CancelIncompleteWatchUpdates(ctx context.Context, watchID string) error
	SetWatchChecked(ctx context.Context, watchID string) error
	SetWatchDeferred(ctx context.Context, watchID string) error
	ListDownstreamWatchIDs(ctx context.Context, watchID string) ([]string, error)

	GetNotificationWatchID(ctx context.Context, notificationID string) (string, error)
	GetPullRequestNotification(ctx context.Context, notificationID string) (*types.PullRequestNotification, error)
	GetWebhookNotification(ctx context.Context, notificationID string) (*types.WebhookNotification, error)
	GetEmailNotification(ctx context.Context, notificationID string) (*types.EmailNotification, error)
	GetSequenceNumberForWatchID(ctx context.Context, watchID string) (int, error)
	GetSequenceNumberForNotificationID(ctx context.Context, notificationID string) (int, error)

	ListReadyImageChecks(ctx context.Context) ([]string, error)
	GetImageCheck(ctx context.Context, imageCheckID string) (*types.ImageCheck, error)
	UpdateImageCheck(ctx context.Context, imageCheck *types.ImageCheck) error

	GetCluster(ctx context.Context, clusterID string) (*types.Cluster, error)
	GetClusterForWatch(ctx context.Context, watchID string) (*types.Cluster, error)
	GetGitHubPathForClusterWatch(ctx context.Context, clusterID string, watchID string) (string, error)

	SetWatchTroubleshootCollectors(ctx context.Context, watchID string, collectors []byte) error
	SetWatchTroubleshootAnalyzers(ctx context.Context, watchID string, analyzers []byte) error
	
	SetWatchLicense(ctx context.Context, watchID string, license []byte) error

	ListReadyAnalysisIDs(ctx context.Context) ([]string, error)
	SetAnalysisStarted(ctx context.Context, supportBundleID string) error
	SetAnalysisFailed(ctx context.Context, supportBundleID string) error
	SetAnalysisSucceeded(ctx context.Context, supportBundleID string, insights string) error
	GetSupportBundle(ctx context.Context, supportBundleID string) (*types.SupportBundle, error)
	GetAnalyzeSpec(ctx context.Context, watchID string) (string, error)
}
