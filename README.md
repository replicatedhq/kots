[![Develop on Okteto](https://okteto.com/develop-okteto.svg)](https://replicated.okteto.dev/deploy?repository=https://github.com/replicatedhq/kots)
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

**NOTE** Currently, the kotsadm pod can **only** be scheduled on nodes with the `linux/amd64` platform.  

### Access the Admin Console
Visit http://localhost:8800 to access the Admin Console, enter the password.

Download the [sample license](https://kots.io/sample-license) for Sentry Pro & upload it to the console. You'll then be presented with configuration settings, preflight checks and other application options.

If you terminate your terminal session, the port-forward will also terminate. To access the admin console again, just run:
```
kubectl kots admin-console --namespace sentry-pro
```

## Supportability

Currently, the KOTS CLI supports OSX (including Apple Silicon arm64) and Linux platforms. However, the Kubernetes resources
that it creates can **only** be scheduled on nodes with the `linux/amd64` platform.

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

## Okteto

### Known issues

1. Kots cannot be installed through the CLI.
2. When a manifest yaml file changes, the only supported way to apply it right now is to redeploy the whole pipeline.

### Unsupported workflows

1. Deploying a vendor application for debugging.  While this could work, it's unsupported, and a different cluster should be used. 

### How To

#### Deploying an application to a different namespace from Kots Admin

If you need to test deploying an application to a different namespace, you'll need to first create the additional namespace in Okteto.
Your permissions will be the same between both namespaces, and you will be able to create deploy/resources there.

##### Use the Kots CLI while Kots Admin is running

1. `okteto up` - Put the the kots pod into dev mode
2. `make build run` - Runs Kots Admin
3. In a new terminal, navigate to the kots project.
4. `okteto exec bash` - Runs bash interactively in the kots pod.
5. `./bin/kots {{COMMAND}}` - Run the kots commands you need.

#### Running KOTS in Helm managed mode in Okteto
Steps to run in Helm managed mode:
1. `okteto pipeline deploy`
1. Ensure your local context is set to your okteto environment
1. Set the `IS_HELM_MANAGED` environment variable for the kots deployment `kubectl set env deployment/kotsadm IS_HELM_MANAGED=true`
1. Remove S3 endpoint: `kubectl set env deployment/kotsadm S3_ENDPOINT=""`
1. Optional:
   - if you wish to use Admin Console with production: `kubectl set env deployment/kotsadm REPLICATED_API_ENDPOINT=""`
   - if you wish to use Admin Console with staging: `kubectl set env deployment/kotsadm REPLICATED_API_ENDPOINT="https://staging.replicated.app"`

### Build V2 (EXPERIMENTAL)

#### Description

This new iteration of our Okteto workspace has significant changes and requires a new workflow by developers.

#### Why

We've been trying to optimize our build times and make developing on Okteto as frictionless as possible.  However, we've realized that there are some fundemental issues with our current strategy, such as:

1. Builds take place in two places (buildkit, in dev containers).  This causes issues with cache sharing, image size, etc.
2. Spike in resources for development containers.  Some of our apps put a heavy strain on resources when built, this require us to either give them a lot of resources while in development mode (which can be long-lasting) or starve them of resources and bottleneck builds.
3. Unable to quickly/easily deploy kubernetes manifest changes.

#### Solution

This V2 work flow attempts to solve these issues by:

1. Build application only on the buildkit servers so that the cache lives in one place and image sizes stay lean.  This excluded applications that have live reloading (web).
2. Only use development containers where needed.  (web apps, schema hero, etc)
3. Update the Okteto manifest to the new schema which allows for separating build and deploy specs, allowing us to run `okteto deploy` and only deploy the manifest. 

#### Reference

| Action               | Syntax | Description                                                                                                |
|----------------------| ------ |------------------------------------------------------------------------------------------------------------|
| Build and Deploy     | `okteto pipeline deploy -f okteto-v2.yml` | Runs both build and deploy sections of the Okteto manifest. Perfect for updating or creating a namespace.  |
| Build single service | `okteto build -f okteto-v2.yml {{SERVICE_NAME}}` | Builds the named service (kotsadm, kotsadm-web, kotsadm-migrations) and pushes it to the Okteto registry. |
| Deploy               | `okteto deploy -f okteto-v2.yml` | Deploys the kubernetes manifests. If there were builds before this command, the new images will be used in the deployment. | 
| Development mode     | `okteto up -f okteto-v2.yml` | Prompts the use for what container to put into development mode.  kotsadm(api), web and migrations will appear for debugging. |

#### Warning

Because this new workflow is experimental, we still have the old workflow in the project.  If you are using the new workflow, and fail to provide the `-f` flag with the v2 manifest, you will be invoking the old workflow.

#### Example workflow: kotsadm change

1. `okteto pipeline deploy -f okteto-v2.yml`
2. Make code changes to kots.
3. `okteto build -f okteto-v2.yml kotsadm`
4. `okteto deploy -f okteto-v2.yml`

#### Example workflow: kubernetes manifest change

1. `okteto pipeline deploy -f okteto-v2.yml`
2. Make manifest changes.
3. `okteto deploy -f okteto-v2.yml`

#### Example workflow: kotsadm web changes

1. `okteto pipeline deploy -f okteto-v2.yml`
2. `okteto up -f okteto-v2.yml`
3. Select kotsadm-web.
4. Make code changes to kotsadm web.
