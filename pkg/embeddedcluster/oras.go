package embeddedcluster

import (
	"context"
	"fmt"

	"go.uber.org/multierr"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

type pullArtifactOptions struct {
	client remote.Client
}

// pullArtifact fetches an artifact from the registry pointed by 'from'. The artifact
// is stored in a temporary directory and the path to this directory is returned.
// Callers are responsible for removing the temporary directory when it is no longer
// needed. In case of error, the temporary directory is removed here.
func pullArtifact(ctx context.Context, srcRepo, dstDir string, opts pullArtifactOptions) error {
	imgref, err := registry.ParseReference(srcRepo)
	if err != nil {
		return fmt.Errorf("parse image reference: %w", err)
	}

	repo, err := remote.NewRepository(srcRepo)
	if err != nil {
		return fmt.Errorf("create repository: %w", err)
	}

	fs, err := file.New(dstDir)
	if err != nil {
		return fmt.Errorf("create file store: %w", err)
	}
	defer fs.Close()

	if opts.client != nil {
		repo.Client = opts.client
	}

	tag := imgref.Reference
	_, tlserr := oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	if tlserr == nil {
		return nil
	}

	// if we fail to fetch the artifact using https we gonna try once more using plain
	// http as some versions of the registry were deployed without tls.
	repo.PlainHTTP = true
	if _, err := oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions); err != nil {
		err = multierr.Combine(tlserr, err)
		return fmt.Errorf("fetch artifacts with or without tls: %w", err)
	}
	return nil
}
