# Remove Postgres dependency and requirement

This proposal is to remove to Postgres service from the Admin Console.

## Goals

Allow the Admin Console to run without Postgres or any new stateful component.
Continued support for running > 1 replica of the KOTS Admin Console API server in "Active-Active" mode.

## Non Goals

Support for configuring applications via kubectl commands (i.e. kubectl edit configmap).

## Background

The KOTS Admin Console currently requires a Postgres database to store various state.
This requires a PVC to be available and we don't currently have a high-avaibility story for this component.
We've had requests to allow external database instances to be used.
Our use of Postgres is very limited, and this proposal recommends a path to removing this requirement completely.

## High-Level Design

Identify and move all state stored in the Postgres data to another existing storage backend.
We have write access to the Kubernetes API, some small data can be stored in config maps, secrets and other K8s objects.
All Kubernetes objects will be native (built-in) types, to perserve the KOTS-doesn't-need-CRD security statement.
We also have an object store, either S3 or an OCI registry for larger objects (application archives and support bundles).
This design is cautious to design for the size limitations of a single object in etcd, while also not creating 1000s of objects in the namespace.

## Detailed Design

Digging into what's stored, the details here focus on the new location for the data currently in the database.
All database tables are defined as [SchemaHero](https://schemahero.io) tables here: https://github.com/replicatedhq/kots/tree/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables.
Many of these tables were used in Ship, but never used in KOTS, while others are critical.

### Tables with data to migrate

| Table Name | Status | Plan |
|------------|--------|------|
| [app](https://github.com/replicatedhq/kots/blob/master/kotsadm/migrations/tables/app.yaml) | Critical | Migration defined in App & App Version |
| [app-downstream-version](https://github.com/replicatedhq/kots/blob/master/kotsadm/migrations/tables/app_downstream_version.yaml) | Critical | Migration defined in App & App Version |
| [app-version](https://github.com/replicatedhq/kots/blob/master/kotsadm/migrations/tables/app_version.yaml) | Critical | Migration defined in App & App Version |
| [cluster](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/cluster.yaml) | Critical | |
| [supportbundle](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/supportbundle.yaml) | Critical | Migration Defined in Support Bundle & Analysis |
| [supportbundle-analysis](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/supportbundle_analysis.yaml) | Critical | Migration Defined in Support Bundle & Analysis |

### Tables to research

| Table Name | Status | Plan |
|------------|--------|------|
| [api-task-status](https://github.com/replicatedhq/kots/blob/master/kotsadm/migrations/tables/api_task_status.yaml) | Used | TBD |
| [app-downstream](https://github.com/replicatedhq/kots/blob/master/kotsadm/migrations/tables/app_downstream.yaml) | | |
| [app-status](https://github.com/replicatedhq/kots/blob/master/kotsadm/migrations/tables/app_status.yaml) | | |
| [kotsadm-params](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/kotsadm_params.yaml) | | |
| [object-store](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/object_store.yaml) | | |
| [pending-supportbundle](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/pending_support_bundle.yaml) | | |
| [preflight-result](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/preflight_result.yaml) | | |
| [preflight-spec](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/preflight_spec.yaml) | | |
| [scheduled-snapshots](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/scheduled_snapshots.yaml) | Used | TBD |
| [session](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/session.yaml) | Used | TBD |
| [ship-user-local](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_user_local.yaml) | Used | TBD |

### Tables without data to drop

| Table Name | Status | Plan |
|------------|--------|------|
| [app-downstream-output](https://github.com/replicatedhq/kots/blob/master/kotsadm/migrations/tables/app_downstream_output.yaml) | Unused | Drop |
| [cluster-github](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/cluster_github.yaml) | Unused | Drop |
| [email-notification](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/email_notification.yaml) | Unused | Drop |
| [feature](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/feature.yaml) | Unused | Drop |
| [github-install](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/github_install.yaml) | Unused | Drop |
| [github-nonce](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/github_nonce.yaml) | Unused | Drop |
| [github-user](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/github_user.yaml) | Unused | Drop |
| [helm-chart](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/helm_chart.yaml) | Unused | Drop |
| [helm-chart-source](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/helm_chart_source.yaml) | Unused | Drop |
| [image-watch](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/image_watch.yaml) | Unused | Drop |
| [image-watch-batch](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/image_watch_batch.yaml) | Unused | Drop |
| [pending-pullrequest-notification](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/pending_pullrequest_notification.yaml) | Unused | Drop |
| [pullrequest-history](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/pullrequest_history.yaml) | Unused | Drop |
| [pullrequest-notification](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/pullrequest_notification.yaml) | Unused | Drop |
| [ship-edit](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_edit.yaml) | Unused | Drop |
| [ship-init](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_init.yaml) | Unused | Drop |
| [ship-init-pending](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_init_pending.yaml) | Unused | Drop |
| [ship-init-pending-user](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_init_pending_user.yaml) | Unused | Drop |
| [ship-notification](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_notification.yaml) | Unused | Drop |
| [ship-output-files](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_output_files.yaml) | Unused | Drop |
| [ship-unfork](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_unfork.yaml) | Unused | Drop |
| [ship-update](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_update.yaml) | Unused | Drop |
| [ship-user](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/ship_user.yaml) | Unused | Drop |
| [track-scm-leads](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/track_scm_leads.yaml) | Unusued | Drop |
| [user-app](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/user_app.yaml) | Unused | Drop |
| [user-cluster](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/user_cluster.yaml) | Unused | Drop |
| [user-feature](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/user_feature.yaml) | Unused | Drop |
| [user-watch](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/user_watch.yaml) | Unused | Drop |
| [watch](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch.yaml) | Unused | Drop |
| [watch-cluster](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch_cluster.yaml) | Unused | Drop |
| [watch-downstream-token](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch_downstream_token.yaml) | Unused | Drop |
| [watch-feature](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch_feature.yaml) | Unused | Drop |
| [watch-license](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch_license.yaml) | Unused | Drop |
| [watch-troubleshoot-analyzer](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch_troubleshoot_analyzer.yaml) | Unused | Drop |
| [watch-troubleshoot-collector](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch_troubleshoot_collector.yaml) | Unused | Drop |
| [watch-version](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/watch_version.yaml) | Unused | Drop |
| [webhook-notification](https://github.com/replicatedhq/kots/blob/00ca0ddbfdfe0811618db04e4f3998c35a4adf34/kotsadm/migrations/tables/webhook_notification.yaml) | Unused | Drop |

### App & AppVersion

The database stores the list of apps that the Admin Console is managing in the app table.
This contains source-of-truth information and also cache for some specs that are in the application archive.

Each app managed by the Admin Console will be managed in a single config map named `kotsapp-<slug>`.
There will be 1 config map per application.
These config maps will be discoverable by KOTS via the inclusion of an annotation: "kots.io/database: app" and "kots.io/appId: <appId>".
This config map will contain keys that loosely map to the current app table:

```yaml
data:
  appSlug: string
  appId: string
  currentSequence: int64
  versions: <stringified JSON array of appversion data (below)>
    ...
```

JSON representation of appversion data to be included above is:

```json
{
    sequence: int64,
    createdAt: timestamp,
    status: enum (Pending, PendingPreflights, Deployed)
}
```

We cannot expect to cache kotsKinds in these "rows" due to size.
We only need the "currentSequence" kotsKinds to be readily accessible to show the config page, dashboard buttons, etc.
Past and pending versions will not have the kotsKinds anywhere execept for in the archive, stored in storage (OCI or S3).
Current version will contain a new config map named `kotsappversion-<slug>`.
This config map will have "kots.io/database: appVersion" and "kots.io/appId: <appId>" and "kots.io/appSequence": <sequence>" annotations.
When deploying a new version, this config map will be replaced with the kots kinds from the new version.
GitOps enabled apps will always have the latest sequence in this config map.

```yaml
data:
  preflight: |
    apiVersion: troubleshoot.replicated.com/v1beta1
    kind: Preflight

  collectors: |
  analyzers: |
  config:
  configValues: |
  application: |
  sigApplication:
  backup: |
  ...
```

### Support Bundle & Analysis

Support bundle archives are stored in the storage location (OCI Registry or S3).
In order to build the list of support bundles on the Troubleshoot tab, we need an index of all support bundles.
This can be converted to a single config map (per app) named `kotssupportbundles-<appId>`.
This config maps will be discoverable by KOTS via the inclusion of an annotation: "kots.io/database: supportbundle" and "kots.io/appId: <appId>"
This config map contains just the IDs and enough info to show the Support Bundle rows (icons and high analysis):

```yaml
data:
  supportBundles: |
    [
        {
            "id": "abc",
            "analysis": {...}
        }
    ]
```

Viewing the full support bundle files will require downloading the bundle from the OCI or S3 storage backend.


## Migration

Current installations will have to be migrated after update.

 TBD

## Alternatives Considered

An embedded database was considered but eliminated because of the requirement to have > 1 replica of the API running at any time.

## Security Considerations

 TBD
