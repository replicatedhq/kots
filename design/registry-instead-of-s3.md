# Optionally replace object store requirement with local registry

KOTS requires an object store in order to store persistent versions of application version artifacts.
KOTS also supports optionally providing a local OCI/Docker image repository to store application container images.
This proposal removes the object store requirement in favor of the image repository.

## Goals

We could remove the requirement to have an object store (currently minio or rook) by using an image repository.
This would be easier to support because there would be less third party components being used.

## Non Goals


## Background

If an image repository is provided for the application, KOTS will create a data image containing the application version manifests and store it in the remote image repository.
For online installs to existing clusters, if a registry is not provided, KOTS will deploy a small, PVC backed image repository.
KOTS will no longer deploy minio or create the object store in Rook by default.

## High-Level Design

When an image repository is set in KOTS, the application manifests will be pushed to the registry instead of the object store.
The code that downloads a version will be able to download a version from the image repository, instead.

## Detailed Design

New CLI flags will be added to `kots install` and `kots pull` to provide a local registry to be used.
When these flags are not provided, KOTS will install Docker Distribution statefulset and service using a ClusterIP.
This will be an insecure registry.


Application manifests are currently packaged as a `.tar.gz` file.
KOTS will continue to do this, but use the [ORAS Go Module](https://github.com/deislabs/oras#oras-go-module) to package these into an image.
The image will be tagged as `registry.hostname` / `registry.namespace` / kots-application-{slug}:{sequence}

For example, if the application slug is `my-app`, and the KOTS Admin Console user provided `registry.somebigbank.com/my-app` as the registry, the first version (sequence 0) will be an ORAS image pushed to `registry.somebigbank.com/my-app/kots-application-my-app:0`.

There are no changes to any YAML schemas or APIs.

Existing installations will be migrated when a version of KOTS with this functionality is started.

kURL installations would work as-is (they already package docker distribution).
In a future release, we could remove the rook object store manifest from kURL install scripts.

## Alternatives Considered

* Bring your own S3/Object Store, as proposed in https://github.com/replicatedhq/kots/pull/474

## Security Considerations

Application manifests may contain sensitive information (secrets).
