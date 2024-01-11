# Building KOTS with apko + melange

## What?

This doc describes a non-production-ready process for building a minimal `kots` image using `melange` and `apko`:

- [`melange`](https://github.com/chainguard-dev/melange) is a tool for reproducibly building APK packages from source
- [`apko`](https://github.com/chainguard-dev/apko) is a tool for reproducibly building container images from APK packages

## Why?

Building with `melange` and `apko` produces smaller, more reproducible images, which can be easier to operate and easier to keep free of vulnerabilities.

## How?

First, build the package from source, using `melange`.

To start, if there isn't already a signing key for the package, we need to generate one:

```sh
melange keygen
```

We only need to build for x86_64, which is faster than building for arm64 since it doesn't require qemu.

```sh
melange build melange.yaml --arch=x86_64
```

> 💡 Only building for your local platform makes builds faster, since it doesn't have to emulate with qemu.
> If you're on an arm64 machine (e.g., Apple Silicon), use `--arch=aarch64` here and below.

Then, build the image from the newly built `kotsadm` package, and the other packages needed by the image, using `apko`:

```sh
apko publish apko.yaml ttl.sh/kotsadm --arch=x86_64
```

This will print the image to stdout, so you can run it:

```sh
docker run $(apko publish ...)
```

### Presubmit GitHub Actions

The above steps are automated in [GitHub Actions](./.github/actions/build-kotsadm-image/action.yml) as a presubmit check for PRs.

The image this workflow produces is only meant for validation, and not meant for production use cases at this time.

## Further Reading

- https://edu.chainguard.dev/open-source/melange/overview/
- https://edu.chainguard.dev/open-source/apko/overview/
