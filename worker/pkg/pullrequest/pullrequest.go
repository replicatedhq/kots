package pullrequest

import (
	"context"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/worker/pkg/store"
	"github.com/replicatedhq/kotsadm/worker/pkg/types"
	"github.com/replicatedhq/kotsadm/worker/pkg/util"
	"github.com/replicatedhq/ship/pkg/state"
	"go.uber.org/zap"
)

type PullRequestRequest struct {
	NewVersionString string

	owner          string
	repo           string
	branch         string
	path           string
	installationID int

	title            string // Used for the PR Title
	message          string // Message of the PR
	commitMessage    string // Commit message for the update
	renderedFileName string // File name of rendered.yaml

	fileName     string // Stored name of rendered file after calling GetRendered method
	fileContents string // Stored contents of rendered file after calling GetRendered method

	SourceBranch string // the branch to make a PR from, defaults to destination branch
}

// NewPullRequestRequest will create a PullRequestRequest object that can be used to create a PR later
// this is separated into a function like this because it's also used in ShouldCreatePullRequest
func NewPullRequestRequest(store store.Store, watch *types.Watch, file multipart.File, owner string, repo string, branch string, path string, installationID int, watchState state.State, title string, sourceBranch string) (*PullRequestRequest, error) {

	newVersionString := ""
	if watchState.V1 != nil && watchState.V1.Metadata != nil && watchState.V1.Metadata.Version != "" {
		newVersionString = watchState.V1.Metadata.Version
	} else {
		newVersion, err := store.GetMostRecentWatchVersion(context.TODO(), watch.ID)
		if err != nil {
			// TODO: log this
			newVersion = &types.WatchVersion{
				VersionLabel: "",
			}
		}
		if newVersion != nil {
			newVersionString = newVersion.VersionLabel
		}
	}

	if len(title) == 0 {
		if newVersionString == "" {
			title = fmt.Sprintf("Update to %s", watch.Title)
		} else {
			title = fmt.Sprintf("Update %s to version %s", watch.Title, newVersionString)
		}
	}

	message := title
	renderedFilename := "rendered.yaml"
	if watchState.V1 != nil && watchState.V1.Metadata != nil {
		if watchState.V1.Metadata.ReleaseNotes != "" {
			message = fmt.Sprintf("Release notes:\n\n%s", watchState.V1.Metadata.ReleaseNotes)
		}
		if watchState.V1.Metadata.Name != "" {
			renderedFilename = fmt.Sprintf("%s.yaml", watchState.V1.Metadata.Name)
		} else if watchState.V1.Metadata.AppSlug != "" {
			renderedFilename = fmt.Sprintf("%s.yaml", watchState.V1.Metadata.AppSlug)
		}
	}

	commitMessage := watch.Title
	if newVersionString != "" {
		commitMessage = fmt.Sprintf("%s - %s", watch.Title, newVersionString)
	}

	fileName, fileContents, err := util.FindRendered(file)
	if err != nil {
		return nil, errors.Wrap(err, "find rendered")
	}

	return &PullRequestRequest{
		NewVersionString: newVersionString,

		owner:          owner,
		repo:           repo,
		branch:         branch,
		path:           path,
		installationID: installationID,

		fileName:     fileName,
		fileContents: fileContents,

		title:            title,
		message:          message,
		commitMessage:    commitMessage,
		renderedFileName: renderedFilename,
		SourceBranch:     sourceBranch,
	}, nil
}

func ShouldCreatePullRequest(logger *zap.SugaredLogger, privateKey string, integrationID int, prRequest *PullRequestRequest) (bool, error) {
	client, err := initGithubClient(integrationID, privateKey, prRequest.installationID)
	if err != nil {
		return false, errors.Wrap(err, "init github client")
	}

	destBranch := prRequest.branch
	if destBranch == "" {
		destBranch = "master"
	}

	pathToRendered := path.Join(prRequest.path, prRequest.renderedFileName)
	file, _, _, err := client.Repositories.GetContents(context.TODO(), prRequest.owner, prRequest.repo, pathToRendered, &github.RepositoryContentGetOptions{
		Ref: destBranch,
	})
	if err != nil {
		return false, errors.Wrap(err, "get file contents")
	}

	fileContents, err := file.GetContent()
	if err != nil {
		return false, errors.Wrap(err, "get content from file")
	}

	if fileContents != prRequest.fileContents {
		return true, nil
	}

	return false, nil
}

func CreatePullRequest(logger *zap.SugaredLogger, privateKey string, integrationID int, prRequest *PullRequestRequest) (int, string, string, error) {
	client, err := initGithubClient(integrationID, privateKey, prRequest.installationID)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "init github client")
	}

	destBranch := prRequest.branch
	if destBranch == "" {
		destBranch = "master"
	}

	sourceBranch := prRequest.SourceBranch
	if sourceBranch == "" {
		sourceBranch = destBranch
	}

	// Get the head SHA
	headRef, _, err := client.Git.GetRef(context.TODO(), prRequest.owner, prRequest.repo, fmt.Sprintf("refs/heads/%s", sourceBranch))
	if err != nil {
		// if the head SHA does not exist, try again with the dest branch
		logger.Warnw("failed to get source branch from sha", zap.Error(err))
		headRef, _, err = client.Git.GetRef(context.TODO(), prRequest.owner, prRequest.repo, fmt.Sprintf("refs/heads/%s", destBranch))
		if err != nil {
			return 0, "", "", errors.Wrap(err, "get head ref")
		}
	}

	// Create a branch for this commit
	branchName := GenerateBranchBame()
	ref := github.Reference{
		Ref:    github.String(fmt.Sprintf("refs/heads/%s", branchName)),
		Object: headRef.GetObject(),
	}
	_, _, err = client.Git.CreateRef(context.TODO(), prRequest.owner, prRequest.repo, &ref)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create branch")
	}

	// Create tree
	treeEntries, err := createTreeEntriesForPullRequest(logger, prRequest)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create tree entries")
	}
	tree, _, err := client.Git.CreateTree(context.TODO(), prRequest.owner, prRequest.repo, fmt.Sprintf("refs/heads/%s", branchName), treeEntries)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create tree")
	}

	// Commit
	parent, _, err := client.Repositories.GetCommit(context.TODO(), prRequest.owner, prRequest.repo, headRef.GetRef())
	if err != nil {
		return 0, "", "", errors.Wrap(err, "get parent commit")
	}
	parentCommit := parent.GetCommit()
	parentCommit.SHA = parent.SHA // This is a weird bug in the github api...

	now := time.Now()
	commit := github.Commit{
		Tree:    tree,
		Message: github.String(prRequest.commitMessage),
		Parents: []github.Commit{
			*parentCommit,
		},
		Author: &github.CommitAuthor{
			Date:  &now,
			Name:  github.String("Replicated Ship"),
			Email: github.String("ship@replicated.com"),
		},
	}
	newCommit, _, err := client.Git.CreateCommit(context.TODO(), prRequest.owner, prRequest.repo, &commit)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create commit")
	}

	ref.Object.SHA = newCommit.SHA
	_, _, err = client.Git.UpdateRef(context.TODO(), prRequest.owner, prRequest.repo, &ref, false)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "attach commit to branch")
	}

	pr := github.NewPullRequest{
		Title: github.String(prRequest.title),
		Head:  github.String(fmt.Sprintf("refs/heads/%s", branchName)),
		Base:  github.String(destBranch),
		Body:  github.String(prRequest.message),
	}

	pullRequest, _, err := client.PullRequests.Create(context.TODO(), prRequest.owner, prRequest.repo, &pr)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create github pull request")
	}

	return pullRequest.GetNumber(), newCommit.GetSHA(), branchName, nil
}

func GenerateBranchBame() string {
	var letters = "abcdefghijklmnopqrstuvwxyz0123456789"

	id := make([]byte, 7)
	for i := range id {
		id[i] = letters[rand.Intn(len(letters))]
	}

	return fmt.Sprintf("ship-%s", id)
}

func createTreeEntriesForPullRequest(logger *zap.SugaredLogger, prRequest *PullRequestRequest) ([]github.TreeEntry, error) {
	entries := make([]github.TreeEntry, 0, 0)

	fullRenderedFilenamePath := strings.Replace(prRequest.fileName, "rendered.yaml", prRequest.renderedFileName, 1)
	entries = append(entries, github.TreeEntry{
		Path:    github.String(strings.TrimPrefix(path.Join(prRequest.path, fullRenderedFilenamePath), "/")),
		Mode:    github.String("100644"),
		Type:    github.String("blob"),
		Size:    github.Int(len(prRequest.fileContents)),
		Content: github.String(prRequest.fileContents),
	})

	return entries, nil
}

func ShouldUsePreviousBranch(previousWatchVersion *types.WatchVersion) bool {
	// // if any of the repo/branch/path/org have changed, don't use the previous branch
	// if notification.RootPath != item.RootPath || notification.Branch != item.Branch || notification.Repo != item.Repo || notification.Org != item.Org {
	// 	return false
	// }

	// // only use the previous branch if the status is unknown or pending - ignored and deployed should both use the target branch
	// if item.GithubStatus != "unknown" && item.GithubStatus != "pending" {
	// 	return false
	// }

	return true
}

func initGithubClient(integrationID int, privateKey string, installationID int) (*github.Client, error) {
	transport, err := ghinstallation.New(http.DefaultTransport, integrationID, installationID, []byte(privateKey))
	if err != nil {
		return nil, errors.Wrap(err, "ghinstallation.new")
	}

	return github.NewClient(&http.Client{Transport: transport}), nil
}
