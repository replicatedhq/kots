package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/google/go-github/v39/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	repoName = "replicatedhq"
	owner    = "kots"
)

func main() {
	srcReleaseTag := os.Getenv("INPUT_SRCRELEASETAG")
	dstReleaseTag := os.Getenv("INPUT_DSTRELEASETAG")

	fmt.Printf("::group::Copying assets\n")
	defer fmt.Printf("::endgroup::\n")

	fmt.Printf("Copying assets from %s to %s\n", srcReleaseTag, dstReleaseTag)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	dstRelease, _, err := client.Repositories.GetReleaseByTag(ctx, repoName, owner, dstReleaseTag)
	if err != nil {
		panic(errors.Wrapf(err, "failed to get %s release", dstReleaseTag))
	}
	deleteAssets(ctx, client, dstRelease)

	srcRelease, _, err := client.Repositories.GetReleaseByTag(ctx, repoName, owner, srcReleaseTag)
	if err != nil {
		panic(errors.Wrap(err, "failed to get source release"))
	}

	copyAssets(ctx, client, srcRelease, dstRelease)
}

func deleteAssets(ctx context.Context, client *github.Client, release *github.RepositoryRelease) {
	assets, _, err := client.Repositories.ListReleaseAssets(ctx, repoName, owner, *release.ID, nil)
	if err != nil {
		panic(errors.Wrapf(err, "failed to list %s release assets", *release.TagName))
	}

	for _, asset := range assets {
		fmt.Printf("Deleting asset %s in release %s\n", *asset.Name, *release.TagName)
		_, err := client.Repositories.DeleteReleaseAsset(ctx, repoName, owner, *asset.ID)
		if err != nil {
			panic(errors.Wrapf(err, "failed to list %s release assets", *release.TagName))
		}
	}
}

func copyAssets(ctx context.Context, client *github.Client, srcRelease *github.RepositoryRelease, dstRelease *github.RepositoryRelease) {
	assets, _, err := client.Repositories.ListReleaseAssets(ctx, repoName, owner, *srcRelease.ID, nil)
	if err != nil {
		panic(errors.Wrapf(err, "failed to list %s release assets", *srcRelease.TagName))
	}

	for _, asset := range assets {
		fmt.Printf("Downloading asset %s from release %s\n", *asset.Name, *srcRelease.TagName)
		fileName := downloadAsset(ctx, client, asset)
		reader, err := os.Open(fileName)
		if err != nil {
			panic(errors.Wrap(err, "failed to open asset file"))
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
			b, _ := ioutil.ReadAll(resp.Body)
			panic(errors.Wrapf(err, "failed to upload %s release asset: %s", *dstRelease.TagName, b))
		}
	}
}

func downloadAsset(ctx context.Context, client *github.Client, asset *github.ReleaseAsset) string {
	reader, _, err := client.Repositories.DownloadReleaseAsset(ctx, repoName, owner, *asset.ID, http.DefaultClient)
	if err != nil {
		panic(errors.Wrapf(err, "failed to get asset %s reader", *asset.Name))
	}
	defer reader.Close()

	writer, err := ioutil.TempFile("", "asset-")
	if err != nil {
		panic(errors.Wrap(err, "failed create temp asset file"))
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		panic(errors.Wrapf(err, "failed create copy asset %s to temp file", *asset.Name))
	}

	return writer.Name()
}
