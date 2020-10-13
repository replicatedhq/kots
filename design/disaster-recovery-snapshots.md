# Optionally enable disaster recovery snapshots

Currently KOTS supports application level snapshots, which can be used to restore a previously deployed application version along with its data.
However in case of a disaster, only application can be restored to a new cluster and Admin Console can never be used to manage it again.
This document proposes a way to implement disaster recovery snapshots using Velero.

## Goals

1. Use Velero to create Admin Console snapshot.
1. Make CLI based disaster recovery possible.
1. Propose a way to enable UI based recovery process.

## Non Goals

1. Improving application deployment process to avoid the `field is immutable` errors.
1. Performing incremental backups of docker images.

## Background

Velero allows a number of ways to include and exclude resources in backups.
In current implememntation, backup is created with the `includedNamespaces` key set to include all namespaces where application is deployed.
All Admin Console resources have label `velero.io/exclude-from-backup: "true"`, which is recognized by Velero to exclude these resources.


Disaster recovery snapshots must include Admin Console data, and potentially all resources.
Due to this requirement, `velero.io/exclude-from-backup` label can no longer be used.
The `includedNamespaces` filter also cannot be used because Admin Console and application can be deployed into the same namespace, which will cause problems during the restore process.

## High-Level Design

Velero backup spec supports the `labelSelector` field.
All resources, application and Admin Console, can be labeled for inclusion in snapshots.
Similarly, restore spec supports the same `labelSelector` field to select specific resources to restore.
Using different `labelSelector` values allows to restore application and Admin Console separately.

## Detailed Design

### Backup

- All resources (Admin Console and application) will have an additional label:
    - `kots.io/backup: velero`
- All application resources will have two additional labels:
    - `kots.io/backup: velero`
    - `kots.io/app-slug: <slug>`
- Backup will be created with `labelSelector` set as in the example below:

        apiVersion: velero.io/v1
        kind: Backup
        spec:
          labelSelector:
            matchLabels:
              kots.io/backup: velero

### Restore

- Admin Console restore will be created with `labelSelector` set as in the example below:

        apiVersion: velero.io/v1
        kind: Restore
        spec:
          labelSelector:
            matchLabels:
              kots.io/kotsadm: "true"

- Application restore will be created with `labelSelector` set as in the example below:

        apiVersion: velero.io/v1
        kind: Restore
        spec:
          labelSelector:
            matchLabels:
              kots.io/app-slug: <slug>

- Restore can be performed using Velero CLI:

        velero restore create --from-backup <backup name> -l kots.io/backup=velero

        velero restore create --from-backup <backup name> -l kots.io/app-slug=<slug>

- `kots install --from-backup` command will be implemented as a wrapper for the above commands
    - `kots restore` is a potential alternative, but the UI-based approached discussed next will also need a `kots install` command.


### Potential blockers

When adding new labels, the next deployment can potentially fail with the `field is immutable` errors.
A workaround is to delete the asset that causes the error prior to deploying the new version.

### UI-based restores

In order to allow a UI-based disaster recovery flow, Admin Console needs to be installed first.
When restoring the Admin Console resources, `kotsadm` pod will be restarted leaving UI in a potentially bad state.
For example, once data is restored, the logged in user's session will be invalidated.
Also receiving restore status will be impossible if the API is not available or session is invalid.


In order to avoid the above problems, the following can be implemented:


1. Only Postgres data, configmaps, and secrets will be backed up (i.e. pods will not have the `kots.io/backup=velero` label).
1. There will be a sidecar pod created (`kotsadm-backup`) that will have the `kots.io/backup=velero` label
    - `kotsadm-backup` will run pre-snapshot hooks to backup Postgres data.
    - `kotsadm-backup` will run post-restore hook to restore Postgres data
    - `kotsadm-backup` will be deleted once backup or restore has completed.
1. `kots install` command will allow installing Admin Console only.


Because Admin Console pods will not be restored, we need to ensure that the version installed is compatible with the the data stored in the snapshot.
To accomplish this, the following will be implemented:


1. All Velero backup objects will have the `kots.io/version` label.
1. During the restore, this label will be used to filter out backups that have non-matching versions.
1. During the restore, users will be informed that a different version of Admin Console needed if they want to restore a different, incompatible backup.


### kURL registry

Airgapped embedded instances will also have images stored in local kURL registry.
These images can be backed up and restored using the `kotsadm-backup` sidecar pod.

## Alternatives Considered

Storing Admin Console in its own separate snapshot has been considered.

## Security Considerations

None