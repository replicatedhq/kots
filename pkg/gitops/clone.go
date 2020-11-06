package gitops

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/pkg/errors"
)

func CloneAndCheckout(workDir string, cloneOptions *git.CloneOptions, branchName string) (*git.Repository, *git.Worktree, error) {
	if cloneOptions.RemoteName == "" {
		cloneOptions.RemoteName = git.DefaultRemoteName
	}

	r, worktree, err := cloneAndCheckoutExisting(workDir, cloneOptions, branchName)
	if err != nil {
		if errors.Cause(err) == transport.ErrEmptyRemoteRepository {
			r, worktree, err = cloneAndCheckoutNew(workDir, cloneOptions, branchName)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to init new repo")
			}
			return r, worktree, nil
		}
		return nil, nil, errors.Wrap(err, "failed to clone existing repo")
	}
	return r, worktree, nil
}

func cloneAndCheckoutExisting(workDir string, cloneOptions *git.CloneOptions, branchName string) (*git.Repository, *git.Worktree, error) {
	r, err := git.PlainClone(workDir, false, cloneOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to clone repo %s", cloneOptions.URL)
	}

	workTree, err := r.Worktree()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get working tree")
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)
	remoteBranchRefName := plumbing.NewRemoteReferenceName(cloneOptions.RemoteName, branchName)

	remoteRef, err := r.Reference(remoteBranchRefName, false)
	if err == plumbing.ErrReferenceNotFound {
		remoteRef, err = r.Head()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get HEAD ref")
		}
	} else if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get ref %s", remoteBranchRefName)
	}

	// check out the branch
	branchRef := plumbing.NewHashReference(branchRefName, remoteRef.Hash())
	err = r.Storer.SetReference(branchRef)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to set ref %s", branchRefName)
	}

	err = workTree.Checkout(&git.CheckoutOptions{
		Create: false,
		Force:  false,
		Branch: branchRefName,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to checkout %s", branchRefName)
	}

	return r, workTree, nil
}

func cloneAndCheckoutNew(workDir string, cloneOptions *git.CloneOptions, branchName string) (*git.Repository, *git.Worktree, error) {
	r, err := git.PlainInit(workDir, false)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to initialize repo %s", cloneOptions.URL)
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: cloneOptions.RemoteName,
		URLs: []string{cloneOptions.URL},
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create remote %s", cloneOptions.URL)
	}

	workTree, err := r.Worktree()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get working tree")
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)

	// check out the branch
	branchRef := plumbing.NewSymbolicReference(plumbing.HEAD, branchRefName)
	err = r.Storer.SetReference(branchRef)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to set ref %s", branchRefName)
	}

	return r, workTree, nil
}
