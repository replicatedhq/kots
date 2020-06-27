# Support for installing KOTS Admin Console in Airgapped Clusters

## Goals

- Install kotsadm to an existing cluster where the cluster does not have internet access
- Install kotsadm to an existing cluster from a workstation where the workstation does not have internet access

## Non Goals

- Create a single installer for kotsadm + application
- Automated installs of kotsadm to existing clusters in airgap mode
- Support private (authenticated) local hosting of kotsadm images
- Support for installing versions of KOTS different from the KOTS CLI version installed

## Background

It's not currently possible to install KOTS Admin Console into an existing cluster that doesn't have outbound internet access.


## High-Level Design



## Detailed Design

Add a `kots admin-console copy-images` to the kots cli to retag and push all kotsadm images to an authenticated registry that is accessible from the cluster.
This new command is used to copy all of the KOTS Admin Console images from one location to another.
The command will support a `--src` and a `--dest` flag, with the `--dest` required.
The `--src` has a default value of the normal KOTS location (index.docker.io/kotsadm/*).

Examples:

To copy images to a local disk (to be brought into a network):

```
kubectl kots admin-console copy-images --dest ./kots-admin-console
```

To copy images from a local disk to an internal registry (assuming already authenticated to registry.somebigbank.com):

```
kubectl kots admin-console copy-images --src ./kots-admin-console --dest docker://registry.somebigbank.com/my-app
```

Next, add a flag to `kots install` to collect / rewrite local registry endpoint/namespace into kots manifests. 
This will assume that the images are already in the destination.
For example:

```
kubectl kots install --kotsadm-registry registry.somebigbank.com/my-app 
```

The only difference between the command above and a "normal" installation is that the `image` tag will be rewritten in all KOTS Admin Console manifests to point to `registry.somebigbank.com/my-app`.
All image names and tags will be unchanged.
For example, the kotsadm-operator will be `registry.somebigbank.com/my-app/kotsadm-operator:v1.17.0`.

Additionally, a new `--yaml` flag will be added to have `kots` only generate the yaml (not apply).

## Testing



## Alternatives Considered



## Security Considerations

- This proposal requires that the locally hosted kotsadm images MUST be accessible from the cluster without authentication for the lifetime of the application

