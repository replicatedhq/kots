[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/replicatedhq/kots)

# Kubernetes Off-The-Shelf (KOTS) Software
Replicated KOTS is the collective set of tools that enable the distribution and management of Kubernetes Off-The-Shelf (KOTS) software. The Kots CLI (a Kubectl plugin) is a general purpose, client-side binary for configuring and building dynamic Kubernetes manifests. The Kots CLI also serves as the bootstrapper for the in-cluster Kubernetes application Admin Console [kotsadm](https://github.com/replicatedhq/kots/tree/main/kotsadm) which can be used to automate the core Kots CLI tasks for managing applications (license verification, configuration, updates, image renaming, version controlling changes, and deployment) as well as additional KOTS tasks (running preflight checks and performing support bundle analysis).

## Distributing a KOTS application
Software vendors can [package their Kubernetes applications](https://docs.replicated.com/vendor/distributing-workflow) or [Helm charts](https://docs.replicated.com/vendor/helm-overview) or [Operators](https://docs.replicated.com/vendor/operator-packaging-about) as a KOTS application in order to distribute the application to cluster operators.

## Kots CLI Documentation
Check out the [full docs on the cluster operator experience](https://docs.replicated.com/reference/kots-cli-getting-started) for using the Kots CLI as a Kubectl plugin.

## Try Kots
Try Kots as a cluster operator by installing the Replicated sample app ([Sentry Pro Example](https://github.com/replicatedhq/kots-sentry/)) into an existing Kubernetes cluster. First, install the Kots CLI (a Kubectl plugin) on your workstation:
```
curl https://kots.io/install | bash
```

### Run `kots install`

The `install` command is the recommended way to learn KOTS. Executing the `install` command will install an application and the [kotsadm](https://github.com/replicatedhq/kotsadm) Admin Console to an existing Kubernetes cluster. This command supports installing Helm charts (without Tiller), standard Kubernetes applications and also Replicated KOTS apps.

Continue with the demo by running the following command:
```
kubectl kots install sentry-pro
```

Set a namespace for the admin console and the application components to be installed, and provide a password for the admin console. After this command completes, the kotsadm Admin Console will be running in your cluster, listening on port :8800 on a ClusterIP service in the namespace you deployed the application to. By default this is exposed to your workstation using kubectl port-forward, but you could set up an ingress/load balancer of your own.

### Access the Admin Console
Visit http://localhost:8800 to access the Admin Console, enter the password.

Download the [sample license](https://kots.io/sample-license) for Sentry Pro & upload it to the console. You'll then be presented with configuration settings, preflight checks and other application options.

If you terminate your terminal session, the port-forward will also terminate. To access the admin console again, just run:
```
kubectl kots admin-console --namespace sentry-pro
```

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
- gomplate (we'll automatically install this for you)
- jq (we'll automatically install this for you using brew if available)

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
