[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/replicatedhq/kots)

# Kubernetes Off-The-Shelf (KOTS) Software
Replicated KOTS is the collective set of tools that enable the distribution and management of Kubernetes Off-The-Shelf (KOTS) software. The Kots CLI (a Kubectl plugin) is a general purpose, client-side binary for configuring and building dynamic Kubernetes manifests. The Kots CLI also serves as the bootstrapper for the in-cluster Kubernetes application Admin Console [kotsadm](https://github.com/replicatedhq/kots/tree/main/web) which can be used to automate the core Kots CLI tasks for managing applications (license verification, configuration, updates, image renaming, version controlling changes, and deployment) as well as additional KOTS tasks (running preflight checks and performing support bundle analysis).

## Getting Started
For installation, usage, and distribution instructions, see the [KOTS documentation](https://docs.replicated.com/intro-kots).

## Supportability

Supports OSX (including Apple Silicon arm64) and Linux platforms.

# Community

For questions about using KOTS, there's a [Replicated Community](https://help.replicated.com/community) forum, and a [#kots channel in Kubernetes Slack](https://kubernetes.slack.com/channels/kots).

# Notifications

By default, KOTS will leverage [MinIO](https://github.com/minio/minio) as a standalone object store instance to store application archives and support bundles. All communication between KOTS and the MinIO object store is limited to a REST API released under the Apache 2.0 license. KOTS has not modified the MinIO source code. Use of [MinIO](https://github.com/minio/minio) is currently governed by the GNU AGPLv3 license that can be found in their [LICENSE](https://github.com/minio/minio/blob/main/LICENSE) file. To remove MinIO usage for this use case in an existing cluster, an optional install flag `--with-minio=false` is available for new [KOTS installs](https://docs.replicated.com/reference/kots-cli-install) or [upgrades from existing versions](https://docs.replicated.com/reference/kots-cli-admin-console-upgrade). To remove MinIO usage for this use case in an embedded cluster, the [`disableS3`](https://kurl.sh/docs/add-ons/kotsadm#advanced-install-options) option is available in the KOTS add-on and can be used for new installs or upgrades.

# Software Bill of Materials
Signed SBOMs for KOTS Go dependencies and are included in each release.
Use [Cosign](https://github.com/sigstore/cosign) to validate the signature by running the following
command.
```shell
cosign verify-blob --key sbom/key.pub --signature sbom/kots-sbom.tgz.sig sbom/kots-sbom.tgz
```

# Development

### Requirements

- MacOS
- Docker Desktop with Kubernetes enabled
- Homebrew

### Running the Development Environment

1. Clone the KOTS repo:
    ```bash
    git clone https://github.com/replicatedhq/kots.git
    cd kots
    ```

1. From the root directory, run:
    ```bash
    make dev
    ```

1. Once the development environment is running, you can access the admin console:
   - Directly at http://localhost:30808
   - Via kURL proxy at http://localhost:30880

### Developing kotsadm web

Changes to the kotsadm web component are reflected in real-time; no manual steps are required.

However, to add, remove, or upgrade a dependency / package:

1. Exec into the kotsadm-web container:
    ```bash
    make kotsadm-web-up
    ```

1. Run the desired `yarn` commands. For example:
    ```bash
    yarn add <package>
    ```

1. When finished, exit the container:
    ```bash
    exit
    ```

### Developing kotsadm API

1. To apply your current changes, run the following commands:
    ```bash
    make kotsadm-up
    ```
    ```bash
    make build run
    ```

1. To apply additional changes, stop the current process with Ctrl+C, then run the following command:
    ```bash
    make build run
    ```

1. When finished developing, run the following command to revert back to the original state:
    ```bash
    exit
    ```
    ```bash
    make kotsadm-down
    ```

### Developing kurl-proxy web / API

1. To apply your current changes, run the following commands:
    ```bash
    make kurl-proxy-up
    ```
    ```bash
    make build run
    ```

1. To apply additional changes, stop the current process with Ctrl+C, then run the following command:
    ```bash
    make build run
    ```

1. When finished developing, run the following command to revert back to the original state:
    ```bash
    exit
    ```
    ```bash
    make kurl-proxy-down
    ```
