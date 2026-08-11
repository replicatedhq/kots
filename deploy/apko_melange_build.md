# Building KOTS with apko + melange

## What?

KOTS packages and images are defined by the SecureBuild Melange and APKO specs:

- [`securebuild/package/melange.yaml`](../securebuild/package/melange.yaml)
- [`securebuild/image/apko-kotsadm.yaml`](../securebuild/image/apko-kotsadm.yaml)
- [`securebuild/image/apko-kotsadm-migrations.yaml`](../securebuild/image/apko-kotsadm-migrations.yaml)
- [`securebuild/image/apko-kurl-proxy.yaml`](../securebuild/image/apko-kurl-proxy.yaml)

Stable releases build these specs with the SecureBuild CLI. Prerelease and CI builds run the same specs locally with `melange` and `apko`:

- [`melange`](https://github.com/chainguard-dev/melange) is a tool for reproducibly building APK packages from source
- [`apko`](https://github.com/chainguard-dev/apko) is a tool for reproducibly building container images from APK packages

## Why?

Building with `melange` and `apko` produces smaller, more reproducible images, which can be easier to operate and easier to keep free of vulnerabilities.

## How?

The local build is implemented by these composite actions:

- [`build-securebuild-package-locally`](../.github/actions/build-securebuild-package-locally/action.yml) adapts the package spec to build an exact source commit with an APK-compatible placeholder version, embeds the requested prerelease version, signs the packages, and uploads the local APK repository.
- [`build-securebuild-image-locally`](../.github/actions/build-securebuild-image-locally/action.yml) downloads that repository and publishes the three APKO images from it.

See the `build-melange-packages`, `build-kotsadm`, `build-migrations`, and `build-kurl-proxy` jobs in [`build-test.yaml`](../.github/workflows/build-test.yaml) for a complete TTL registry example.

### Presubmit GitHub Actions

The above steps are automated in GitHub Actions as a presubmit check for PRs.

The image this workflow produces is only meant for validation, and not meant for production use cases at this time.

## Further Reading

- https://edu.chainguard.dev/open-source/melange/overview/
- https://edu.chainguard.dev/open-source/apko/overview/
