# Support for installing KOTS Admin Console in Airgapped Clusters

## Goals

- Install kotsadm to an existing cluster where the cluster does not have internet access
- Install kotsadm to an existing cluster from a workstation where the workstation does not have internet access at the time of installation
- Support different credentials for push / pull (workstation / cluster) to the local registry
- Support temporary/dynamic and persistent/static credentials for pushing kotsadm images to the registry
- Support only persistent/static credentials for pulling kotsadm images from the airgapped registry
- Support different credentials for pushing vs pulling images to the local registry

## Non Goals

- Create a single installer for kotsadm + application
- Create a single download for kotsadm, kots plugin, application and license
- Unattended, automatic installs of kotsadm to existing clusters in airgap mode
- Support for installing versions of KOTS different from the KOTS CLI version installed

## Background

It's not currently possible to install KOTS Admin Console into an existing cluster that doesn't have outbound internet access.

## High-Level Design

Airgap Workstation: the computer with kubectl access to the airgap cluster
Public Workstation: the computer with access to the internet

From the public workstation:
1. The user downloads the kots plugin suitable for Airgap Workstation.
1. The user downloads the kotsadm archive containing kots container images
1. The user downloads the airgap bundle of the application
1. The user copies the kots plugin, the kotsadm archive and the application airgap bundle onto a medium (eg a USB stick) that can be transfered into the airgapped network.

From the airgap workstation:
1. The user will install the kots plugin 
1. The user will run a `kubectl kots` command to copy the container images in the kotsadm archive to the local registry, providing registry details for pushing
1. The user will install kotsadm to the cluster using the kots plugin via `kots install` passing local registry information which will be used to generate the kotsadm yaml and image pull secrets (if neccessary)
1. The user will install the application and license using the existing airgap installation process 

## Detailed Design

The CI process, on tag, will create the kotsadm archive and include it in the GitHub Release.
The filename will be `kots_airgap_<version>.tar.gz` and this file contains .tar file exports of all kotsadm images.
The files SHOULD be stored in the archive in the same format as application airgap files are stored.

Add a `kots admin-console push-images` to the kots cli to retag and push all kotsadm images to an authenticated registry that is accessible from the cluster, using the desired push credentials.
This new command is used to copy all of the KOTS Admin Console images from one location to another.

Example:

```
kubectl kots admin-console push-images ./kotsadm-1.16.1.tar.gz registry.somebigbank.com/my-app \
  --registry-username push-user \
  --registry-password abcdef
```

Stretch: `--registry-username` and `--registry-password` are both optional.
If not present, attempt to authenticate using the already authenticated local docker credentials.
Stretch: if still not authenticated, prompt for username/password, use these to push but do not save locally.
Stretch: if username is specified and password is not, prompt for password to use, do not save locally.

KOTS will then load the images from the file in the first parameter, retag them to the local registry and push. 
All images will retain the same image name and tag, just the registry, namespace will change.

At this point, the images are in the registry, but not running in the cluster yet.
A workflow could scan these images before continuing.

Next, add a flag to `kots install` to collect / rewrite local registry endpoint/namespace into kots manifests, using the desired pull credentials.
This will assume that the images are already in the destination.

For example:

```
kubectl kots install --kotsadm-registry registry.somebigbank.com/my-app \
  --registry-username pull-user \
  --registry-password fedcba \
  app-name
```

If a `registry-username` and `registry-password` are provided, an `imagePullSecret` will be created and deployed with the registry credentials and added to all kotsadm pods.

The only difference between the command above and an online installation is that the `image` tag will be rewritten in all KOTS Admin Console manifests to point to `registry.somebigbank.com/my-app` and an `imagePullSecret` MAY be added.
All image names and tags will be unchanged.
For example, the kotsadm-operator will be `registry.somebigbank.com/my-app/kotsadm-operator:v1.16.1`.

Stretch: Additionally, a new `--yaml` flag will be added to have `kots` only generate the yaml (not apply) to support gitops workflows.

## Limitations

- No support for custom branding in airgap installations to existing clusters
- No support for `requireMinimalRBACPrivileges` set to `true` (all installs will be cluster admin, which matches all use cases of KOTS install today)

## Assumptions

- The airgap workstation will have kubectl access to the cluster
- The airgap workstation must have push access to an image registry that the cluster has pull access 
- The cluster credentials to pull from the local registry is either unauthenticated or static and persistent credentials (i.e. no Vault integration or other dynamically generated credentials will be supported)

## Testing

TBD

## Alternatives Considered



## Security Considerations

- No open security considerations
