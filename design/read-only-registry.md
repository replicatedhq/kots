# Support for Read-Only Registries

## Goals

- Configure kotsadm to use application images from a private registry which kotsadm only has read-only access to (images must be installed or copied to the private registry outside of KOTS)
- Support using read-only registry during initial installation
- Support using read-only registry on application updates
- Support using read-only registry when settings are modified on the Registry Settings page in Admin Console
- Support read-only registries in airgap and on-line modes

## Non Goals

- Delivering images to the read-only registry.
- Creating imageless airgap bundles.
- Addressing existing inconsistencies and bugs in multi-app support.

## Background

Some production environments follow strict processes to control which images are allowed in and subsequently run.
Images are usually vetted in a different staging environment, often with a combination of automatic and manual approvals.
Then through some strict change management process, are pulled into or copied to the production environment.
Applications running in the production environment are only given read-only credentials to consume those images.
Right now, KOTS will always push images to the private registry, making it impossible to use in such an environment.

## Detailed Design

### Install

A new command line flag will be added: `--disable-image-push`.
This flag will be app specific since registry settings are tied to the application.

#### Online Admin Console

This is the most common use of kots install command.
It results in kotsadm being installed in online mode, while the application can be either online or airgapped:

```
kubectl kots install app-slug --disable-image-push
```

When the above command is used, the user will go through the existing installation flow without any changes from the user's perspective.
Internally, the functionality will change as follows:
- If airgap option is selected, images will **not** be pushed to the private registry specified in the UI.
- If online option is selected, updating Registry Settings page will **not** result in images being pushed.

The Registry Settings page is discussed in more detail below.

#### Airgapped Admin Console

In this case, Admin Console images are pulled from a private registry, while the application can be either online or airgapped:

```
kubectl kots install app-slug \
  --kotsadm-registry my.private.registry/myapp \
  --registry-username <username> \
  --registry-password <password> \
  --disable-image-push
```

When the above command is used, the following will happen:

1. All kotsadm pods will be rewritten to pull images from the private registry. This is existing functionality.
1. Provided registry credentials will be saved in `kotsadm-private-registry` secret in the app's namespace. This is existing functionality.
1. The `kots admin-console push-images` command can be ommitted.
1. During license upload in Admin Console UI, registry settings will be populated with these values.
1. After the application is installed, Registry Settings page will be populated with these values.

During application installation:
- If airgap option is selected, images will **not** be pushed to the private registry specified in the UI.
- If online option is selected, updating Registry Settings page will **not** result in images being pushed.

#### Fully airgapped mode

In this case, both Admin Console and application are installed in airgapped mode:

```
kubectl kots install app-slug \
  --kotsadm-registry my.private.registry/myapp \
  --registry-username <username> \
  --registry-password <password> \
  --disable-image-push \
  --airgap-bundle /path/to/app.airgap \
  --license-file /path/to/license.yaml
```

When the above command is used, CLI will skip pushing images and proceed with the rest of the instalaltion.
User will be redirectd to the Config screen.
Other command line arguments can be used for headless and/or unattended install experience without any additional changes.

### Updates

#### Online Admin Console

Upgrading Admin Console installed in online mode does not change.

#### Airgapped Admin Console

Upgrading Admin Console installed in airgapped mode has to be done with the `--disable-image-push` flag specified on the command line:

```
kubectl kots admin-console upgrade \
  --kotsadm-registry my.private.registry/myapp \
  --registry-username <username> \
  --registry-password <password> \
  --disable-image-push
```

In this case, the `kots admin-console push-images` command can be ommitted as well.

#### Online application without private registry

Upgrading online application without private registry does not change.

#### Online application with private registry

Upgrading online application with private registry will require the user to push images to the private registry out of band.
Admin Console will still create a new version downloaded from the server and rewrite all images to be pulled from the private registry.

Internally the functionality changes as follows:
- Images will not be pushed to private registry.

#### Airgapped application (UI)

Upgrading airgapped application using Admin Console UI will require the user to push images to the private registry out of band.
Airgap bundle still has to be uploaded through the UI.
Admin Console will still create a new application version and rewrite all images to be pulled from the private registry.

Internally the functionality changes as follows:
- Images will not be pushed to private registry.

#### Airgapped application (CLI)

Upgrading airgapped application using Admin Console CLI will require the user to push images to the private registry out of band.
Admin Console will still create a new application version and rewrite all images to be pulled from the private registry.

```
kubectl kots upstream upgrade app-slug \
  --airgap-bundle /path/to/app.airgap \
  --kotsadm-namespace myapp \
  --kotsadm-registry my.private.registry \
  --registry-username <username> \
  --registry-password <password> \
  --disable-image-push
  ```

Internally the functionality changes as follows:
- Images will not be pushed to private registry.

### Registry Settings page

Registry settings page will have a checkbox named `Disable Image Push`.

#### Online application installs

In online installs this checkbox is can be toggled freely.

- When the checkbox is unchecked, saving registry works without any changes.
- When checkbox is checked, saving registry settings will not push images.

#### Airgapped installs

In airgapped environments, changing registry host and namespace is not supported today.
Therefore this checkbox will reflect the value of the `--disable-image-push` flag used to install latest app version.

### Preflight checks

A new preflight check will be added to validate that images are present in their respective registries.
This will work automatically and use the same logic to find images that is currently used to automatically rewrite images.

## Design Assumptions

- The end customer will provide an external mechanism for loading images into the private registry
- There is no expectation of additional restrictions (such as Kubernetes RBAC) on binaries that may need to run and interact with the production cluster (such as kots, support-bundle, and velero)
- There is no expectation of additional security restrictions in production, such as iptable rules, SELinux rules, ufw/firewalld services, that may restrict other existing KOTS functionality in a production environment

## Security Considerations

- No open security considerations
