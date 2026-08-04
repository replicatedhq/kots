# Building KOTS packages and images

## Spec files

| Component | Melange | Apko |
|---|---|---|
| kotsadm (+ kotsadm-migrations, kurl-proxy subpackages) | `deploy/melange.yaml` | `deploy/apko.yaml` |
| kotsadm-migrations image | — | `migrations/deploy/apko.yaml` |
| kurl-proxy image | — | `kurl_proxy/deploy/apko.yaml` |

A single melange spec (`deploy/melange.yaml`) builds three APK packages: `kotsadm`, `kotsadm-migrations`, and `kurl-proxy` (the latter two as subpackages). Three separate apko specs build the corresponding container images from those packages.

## Production builds (SecureBuild)

Production builds are triggered by pushing a semver tag (`v*.*.*`). The `release.yaml` workflow calls the SecureBuild CLI to build and publish packages, then images:

```
securebuild build package --package-family-name kotsadm --tag <version>
securebuild build image   --image-name kotsadm             --tag <version>
securebuild build image   --image-name kotsadm-migrations  --tag <version>
securebuild build image   --image-name kurl-proxy          --tag <version>
```

The melange spec's `git-checkout` step clones the tagged commit, and SecureBuild overrides `package.version` with the `--tag` value. Packages are published to `https://apk.cve0.io` and images are pushed to Docker Hub.

A standalone `publish-securebuild.yml` workflow is also available for manual rebuilds via `workflow_dispatch`.

## Pre-release builds (local melange + apko)

Nightly, alpha, and PR builds run melange and apko locally in GitHub Actions. Since there is no tag to check out, the `git-checkout` pipeline step is stripped from the melange spec at runtime (using `yq`), and melange builds from the local checkout via `--source-dir`.

The local build flow (automated in `build-custom-melange-package` and `build-custom-image-with-apko` composite actions):

```sh
# 1. Strip git-checkout from melange spec and set package.version to the git tag
yq 'del(.pipeline[] | select(.uses == "git-checkout"))' deploy/melange.yaml > melange-local.yaml
yq -i ".package.version = \"$(echo $GIT_TAG | sed 's/^v//')\"" melange-local.yaml

# 2. Build all packages (per arch)
melange build melange-local.yaml --arch=x86_64 --source-dir . --signing-key melange.rsa

# 3. Add local packages repo + keyring to apko spec
yq '.contents.repositories += ["./packages/"]' deploy/apko.yaml > apko-local.yaml
yq '.contents.keyring += ["./melange-amd64.rsa.pub", "./melange-arm64.rsa.pub"]' -i apko-local.yaml

# 4. Build and push image
apko publish apko-local.yaml <registry>/<image>:<tag> --arch=x86_64,arm64
```

Repeat step 3-4 for each apko spec (`deploy/apko.yaml`, `migrations/deploy/apko.yaml`, `kurl_proxy/deploy/apko.yaml`).

### Workflows

- **`alpha.yaml`** — nightly builds on `main` branch push; pushes images with `:alpha` tag
- **`build-test.yaml`** — PR builds; pushes images to `ttl.sh` for validation
- **`release.yaml`** — tag pushes use SecureBuild; branch pushes use local melange/apko with nightly tags
