# Automated Airgap Installations and Updates

Automated installs in KOTS are limited to online installs only.
It's not possible to automate an end-to-end installation of an application in airgapped mode.
By adding automation of airgapped applications, we can also enable a workflow for installing applications to airgapped (existing) clusters.

## Goals

- Automated installs of KOTS (Admin Console) and the application to an existing airgapped cluster
-

## Non Goals

- Airgap packaging for non-Replicated applications
- Distribution of the KOTS CLI as part of the application
- Support for environments without a local registry
- Support for non-Linux based container runtimes

## Background

See top section.


## High-Level Design

This proposal changes the `kots pull` command to download the `.airgap` bundle from the Replicated servers, unless a path to an airgap bundle is provided on the command line with the `--airgap-package` flag.
The `--license-file` parameter will be required.
A local registry will be required, and it must be accessible from the workstation that is orchestrating this process.
KOTS pull will load the KOTS images into the registry, and write out the application manifests for KOTS (Admin Console).
These manifests can be applied using `kubectl`, and include the license file that was supplied, in addition to `--config-values` that were optionally supplied.
A new automation flag will be passed into the automation license file, via an annotation, that the installation should be completed as airgapped.
The airgap bundle will be pushed to the local registry, and read from the Admin Console automation.

## Detailed Design

The end result here will make the following completely automate an installation where a cluster does not have access to the internet:

```shell
kots pull \
  --license-file ./license.yaml \
  --config-values ./config.yaml \ # optional, needed to automate config, if there is app config
  --airgap \ # required to select airgap mode
  --airgap-package ./app.airgap \ # optional path to airgap bundle. if not provided, it will be downloaded from the replicated api server
  --registry-endpoint registry.myco.com \
  --registry-namespace my-app \ # the namespace (org) in the registry to use
  --registry-username botaccount \ # required if the local dockerconfig is not already authenticated
  --registry-password asdbc \ # required to automate if not provided in the local dockerconfig. if not provided, the CLI will prompt
  app-slug # the application slug
```

When the above command is run, the following happens:
1. License file is synced with the server (exchanged for an updated version) and validated locally
2. If `--airgap` is set and `--airgap-package` is not provided, the airgap package is downloaded from replicated.app
3. The airgap package is validated
  a. checksum
  b. signature
  c. match app in license
4. The KOTS images are downloaded for the matching version of KOTS (if the CLI is 1.15.1, the Admin Console 1.15.1 is used)
  a. this will always pull the linux images today
5. KOTS application images are retagged and pushed to the registry provided in the CLI
6. The airgap bundle is wrapped into an OCI image using ORAS and pushed to the registry with a predefined name/tag (TBD)
7. Admin Console yaml is written to disk in ./app-slug/base/*
8. A kustomize patch is created in ./app-slug/overlays/{registry-endpoint}/kustomization.yaml to change the images and inject image pull secrets
9. The application automation patch (license, config, etc) are written to ./app-slug/overlays/automation/kustomization.yaml (referring to #7 as base)
  a. the automation annotation should include the image pull secret name and the application bundle image/tag from #6
9. Instructions are provided to run `kubectl apply -k ./app-slug/overlays/local/kustomization` to continue

When KOTS starts, the automation annotations on the KOTS container will load the license, config values, and pull the application airgap from the registry.

## Alternatives Considered


## Security Considerations

Will it be expected that the local image registry credentials are automatically pass into the cluster, if not overridden on the CLI?
This could be more elevated permissions than desired.
