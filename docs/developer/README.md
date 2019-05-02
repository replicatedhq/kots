# Developer Docs

These are docs to set up a local development environment of Ship Cluster.

This has been tested on Docker for Mac with Kubernetes installed.

You need:

1. Docker for Mac
2. Docker's built-in Kubernetes distribution enabled
3. kubectl context set to `docker-for-desktop`
4. Skaffold installed
5. Kustomize installed

Then, right from the root of this repo, run:

```
skaffold dev
```
