package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	repoName = "replicatedhq"
	owner    = "kots"
)

func main() {
	fmt.Printf("::group::Copying assets\n")
	defer fmt.Printf("::endgroup::\n")

	for i := 0; i < 3; i++ {
		err := attemptAssetCopy()
		if err == nil {
			return
		}

		fmt.Printf("Copy assets attempt %d failed: %v\n", i+1, err)
		time.Sleep(10 * time.Second)
	}

	panic(errors.New("all retries failed"))
}

func attemptAssetCopy() error {
	srcReleaseTag := os.Getenv("INPUT_SRCRELEASETAG")
	dstReleaseTag := os.Getenv("INPUT_DSTRELEASETAG")

	fmt.Printf("Copying assets from %s to %s\n", srcReleaseTag, dstReleaseTag)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	dstRelease, _, err := client.Repositories.GetReleaseByTag(ctx, repoName, owner, dstReleaseTag)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s release", dstReleaseTag)
	}
	err = deleteAssets(ctx, client, dstRelease)
	if err != nil {
		return errors.Wrapf(err, "failed to delete %s release assets", dstReleaseTag)
	}

	srcRelease, _, err := client.Repositories.GetReleaseByTag(ctx, repoName, owner, srcReleaseTag)
	if err != nil {
		return errors.Wrap(err, "failed to get source release")
	}

	return errors.Wrap(copyAssets(ctx, client, srcRelease, dstRelease), "failed to copy assets")
}

func deleteAssets(ctx context.Context, client *github.Client, release *github.RepositoryRelease) error {
	assets, _, err := client.Repositories.ListReleaseAssets(ctx, repoName, owner, *release.ID, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to list %s release assets", *release.TagName)
	}

	for _, asset := range assets {
		fmt.Printf("Deleting asset %s in release %s\n", *asset.Name, *release.TagName)
		_, err := client.Repositories.DeleteReleaseAsset(ctx, repoName, owner, *asset.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to list %s release assets", *release.TagName)
		}
	}

	return nil
}

func copyAssets(ctx context.Context, client *github.Client, srcRelease *github.RepositoryRelease, dstRelease *github.RepositoryRelease) error {
	assets, _, err := client.Repositories.ListReleaseAssets(ctx, repoName, owner, *srcRelease.ID, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to list %s release assets", *srcRelease.TagName)
	}

	for _, asset := range assets {
		fmt.Printf("Downloading asset %s from release %s\n", *asset.Name, *srcRelease.TagName)
		fileName, err := downloadAsset(ctx, client, asset)
		if err != nil {
			return errors.Wrap(err, "failed to download asset")
		}

		reader, err := os.Open(fileName)
		if err != nil {
			return errors.Wrap(err, "failed to open asset file")
		}
		defer reader.Close()

		opts := &github.UploadOptions{}
		if asset.Name != nil {
			opts.Name = *asset.Name
		}
		if asset.Label != nil {
			opts.Label = *asset.Label
		}
		if asset.ContentType != nil {
			opts.MediaType = *asset.ContentType
		}

		fmt.Printf("Uploading asset %s to release %s\n", *asset.Name, *dstRelease.TagName)
		_, resp, err := client.Repositories.UploadReleaseAsset(ctx, repoName, owner, *dstRelease.ID, opts, reader)
		if err != nil {
			var b []byte
			if resp != nil && resp.Body != nil {
				b, _ = io.ReadAll(resp.Body)
			}
			return errors.Wrapf(err, "failed to upload %s release asset: %s", *dstRelease.TagName, b)
		}
	}

	return nil
}

func downloadAsset(ctx context.Context, client *github.Client, asset *github.ReleaseAsset) (string, error) {
	reader, _, err := client.Repositories.DownloadReleaseAsset(ctx, repoName, owner, *asset.ID, http.DefaultClient)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get asset %s reader", *asset.Name)
	}
	defer reader.Close()

	writer, err := os.CreateTemp("", "asset-")
	if err != nil {
		return "", errors.Wrap(err, "failed create temp asset file")
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return "", errors.Wrapf(err, "failed create copy asset %s to temp file", *asset.Name)
	}

	return writer.Name(), nil
}
