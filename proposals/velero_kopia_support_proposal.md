## Executive Summary

kURL and KOTS supports Velero up to version 1.16, which supports Kubernetes below 1.34.  Customers with Kubernetes versions 1.34+ do not have a tested, supported version of Velero to use with KOTS/kURL.  Replicated will stop support for Kubernetes 1.33 at the end of June 2026\.

Velero **1.17 made Kopia the only file-system backup (FSB) uploader** and removed the `--uploader-type=restic` install flag. KOTS currently hard-codes Restic in its install instructions, e2e tests, and generated `BackupStorageLocation` configs. Without changes, customers who install or upgrade to Velero 1.17+ will receive invalid install commands and snapshots will fail.

With KOTS in an existing cluster, we do not control the updates and it is feasible that customers are running 1.17+ with KOTS, and just don't know that the backups are failing due to a lack of support for kopia.

This proposal recommends making KOTS **version-aware** so that it emits the correct Velero install flags and `BackupStorageLocation` config for the installed Velero version. **Restic support is not being removed.** Existing Restic backups remain restorable, and customers can upgrade Velero one minor version at a time at their own pace.

## Scope

- **Velero versions:** 1.10.1 through **1.18.x** (the currently supported range). Velero **1.19 is out of scope** for now.  
  - 1.19 will remove customer's ability to restore Restic backups, and is out of scope for this reason.  
  -   
- **KOTS, kURL, LVP plugin, and docs** are all in scope because each touches the install experience or compatibility claims.  
- **No migration** of existing Restic backups.  
- We do not enforce Velero and Kubernetes version coordination, it remains possible to install unsupported combinations.

## Customer problem and why this matters

- A customer on Velero 1.16 who later upgrades to 1.17 will find that KOTS-generated install instructions no longer work (`--uploader-type=restic` is rejected), and if they upgrade disregarding the instructions, they will fail to make backups.  
- A customer on kURL cannot select Velero 1.17 or 1.18 because those add-on versions are not available.  
- Our public docs still say KOTS supports only Restic  
- Without this work, KOTS appears to lag behind a supported upstream dependency, blocking security and bug-fix upgrades for customers who rely on snapshots.  
  - Velero 1.16 is only tested up to Kubernetes 1.33

## Customer interaction and upgrade path

### Customers upgrading KOTS but not Velero

- **No action is required** when upgrading KOTS alone. KOTS will continue to detect the installed Velero version and generate the correct flags for that version.  
- **Restic support is not planned for removal.** Because customers keep older installations running for a long time, KOTS will keep the Restic fallback code and documentation.  
- Kubernetes version limitations will be documented on the docs site.

### Customers who upgrade Velero to 1.17+

- **Restic backups remain restorable.** Velero 1.17 and 1.18 can still restore from Restic repositories, and KOTS will preserve the `resticRepoPrefix` config on existing Restic-era `BackupStorageLocation` objects.  
- **KOTS does not delete or migrate Restic repositories.** New backups on Velero 1.17+ will use Kopia repositories alongside the existing Restic data, **provided the BackupStorageLocation is compatible with Kopia**.  
- **New backups use Kopia automatically.** The first scheduled or manually triggered backup after the Velero upgrade will create Kopia repositories. There is no need to force a new backup.  
- **LVP BSLs are an exception.** If the existing BSL uses the LVP plugin (`replicated.com/hostpath`, `replicated.com/nfs`, or `replicated.com/pvc`), Kopia cannot create a repository against it. The customer must reconfigure to a Kopia-compatible BSL (e.g., S3-compatible storage) for new FSB backups. Existing Restic backups remain restorable.  
- **Customer communication:** Show a non-blocking informational banner in the Admin Console when Velero 1.17+ is detected and Restic snapshots exist, explaining that new backups will use Kopia and that existing Restic backups are still restorable. Also document the behavior in the upgrade and snapshot docs.  
- **Pod volume discovery is unchanged.** KOTS already uses the `backup.velero.io/backup-volumes` annotation to opt volumes into file-system backup. This annotation is uploader-agnostic and works for both Restic and Kopia.  
- Velero upgrades are only supported one minor version at a time. A customer on 1.16 can go to 1.17, then 1.18, using KOTS guidance.  
- KOTS will show the correct install command for each version:  
  - **Velero \< 1.10:** `--use-restic --use-volume-snapshots=false`  
  - **Velero 1.10–1.16:** `--use-node-agent --uploader-type=restic --use-volume-snapshots=false`  
  - **Velero 1.17+:** `--use-node-agent --use-volume-snapshots=false` (Kopia is the default)  
- For customers who want Kopia on 1.10–1.16, the `--uploader-type=kopia` flag can be used explicitly, but KOTS will default to the upstream default for each version to avoid surprises.  
- To display the right instructions, we will need to know the version to be installed.  We provide instructions prior to the installation, so will have to just **ask an additional question** in order to get the correct configuration.

### Upgrade process from a customer point of view

KOTS detects whether Velero is already installed by checking for the `velero` Deployment and the node-agent DaemonSet in the `velero` namespace. When an existing installation is detected, KOTS shows **upgrade instructions** instead of a fresh-install command.

For a customer on 1.16 who wants to upgrade to 1.17+:

1. **Install the Velero 1.17+ CLI.**  
2. **Update the Velero CRDs:**

```shell
velero install --crds-only --dry-run -o yaml | kubectl apply -f -
```

3. **Remove or change the stale `--uploader-type=restic` flag** in the Velero deployment and node-agent DaemonSet. The official Velero docs suggest replacing it with `--uploader-type=kopia`:

```shell
kubectl get deploy -n velero -ojson \
  | sed 's/"--uploader-type=restic"/"--uploader-type=kopia"/g' \
  | kubectl apply -f -
```

4. **Update the Velero and plugin images** to the target 1.17.x/1.18.x version.  
5. **Confirm the upgrade** using the `velero version` command.

KOTS will not automate these steps, but it will surface the correct version-aware instructions and a link to the official Velero upgrade docs. **No BSL reconfiguration is required for Restic restores**: existing Restic backups remain restorable. **If the existing BSL uses LVP (HostPath/NFS/PVC), a new Kopia-compatible BSL is required for new FSB backups** because Kopia cannot use the LVP plugin.

After the upgrade, if the customer re-configures the storage destination through KOTS for Velero 1.17+, KOTS will generate an S3-compatible BSL config (e.g., the internal Minio path) instead of an LVP-backed config.

### New customers

- New installs will use whatever Velero version they provide. KOTS will generate the correct command and storage config for that version.  
- The Admin Console destination picker (AWS, GCP, Azure, S3-compatible, Internal, NFS, HostPath) does not change for destination selection, but for Velero 1.17+ the HostPath/NFS/PVC options are not available for new file-system backups because the LVP plugin is not compatible with Kopia. KOTS should warn or disable those options when Velero 1.17+ is selected and route the customer to S3-compatible storage for local FSB.  
- For Velero 1.17+, KOTS will use the S3-compatible Minio internal-storage path (`provider: aws` with `s3Url`) for local/internal FSB instead of LVP.
## What needs to change, and where

### What is the Replicated LVP plugin and how is it installed?

The `replicated/local-volume-provider` Velero plugin is an object-store plugin that lets Velero use a local filesystem path as a `BackupStorageLocation`.

**How it is installed:**

When KOTS (or the user) runs `velero install --plugins docker.io/replicated/local-volume-provider:<tag>`, Velero adds the plugin image as an **init container** to the Velero deployment. The init container copies the plugin binary into the `/plugins` emptyDir volume. The Velero server container then loads it at startup. This is the same mechanism used for all Velero plugins (AWS, GCP, Azure, etc.).

**Where it is used:**

It is used **only** for internal storage destinations chosen in the Admin Console:

- **HostPath**
- **NFS**
- **PVC** (internal)

It is **not** used for AWS S3, GCP, Azure, or other S3-compatible object stores.

When the customer selects one of these internal destinations, KOTS generates the `BackupStorageLocation` config that points Velero at the LVP plugin and includes the local filesystem path. The plugin is loaded by Velero as an init container, patches the Velero deployment and node-agent DaemonSet to mount the filesystem path, and then serves object-store requests on behalf of Velero.

### 1\. KOTS repo (`replicatedhq/kots`)

| File | Change |
| :---- | :---- |
| `pkg/print/velero.go` | Make CLI and UI instructions version-aware. Use the flags listed above. If a Velero deployment is already present, show upgrade instructions (link to Velero docs + flag-change note) instead of the fresh-install command. |
| `web/src/components/snapshots/ConfigureSnapshots.jsx` | Add `isVelero17OrNewer()` helper; return the correct flags based on `snapshotSettings.veleroVersion`. |
| `e2e/velero/cli.go` | Remove the hard-coded `--uploader-type=restic`; use the version-appropriate flag. |
| `e2e/playwright/regression/shared/cli.ts` | Replace the single `isVelero10OrNewer` flag branch with a version-aware helper. |
| `e2e/scripts/deps.sh` | Move from a hard pin to `v1.16.2` to a parameterized/matrix approach so CI can run against both `v1.16.x` and `v1.18.x`. |
| `pkg/snapshot/store.go` | For Velero 1.17+, do not create new BSLs with LVP providers (`replicated.com/hostpath`, `replicated.com/nfs`, `replicated.com/pvc`). Use S3-compatible storage for local/internal FSB. Preserve `resticRepoPrefix` on existing Restic-era BSLs. See the explicit detection rule below. |
| `cmd/kots/cli/velero.go` | Remove the unused `resticRepoBase` constant. |
| `pkg/kotsadmsnapshot/logparser_test.go` | Add Kopia/Velero 1.17 log fixtures; keep Restic fixtures for regression coverage. |

#### How KOTS decides whether a BSL is Restic-era

KOTS uses the **existing BSL object** as the source of truth, not the target Velero version alone:

- **When re-configuring a storage destination:** KOTS reads the existing BSL config from the cluster first.
  - If the existing BSL already has a `resticRepoPrefix` key, **preserve it** on the updated BSL. The BSL remains a Restic-era repository and can still be restored from on Velero 1.17+.
  - If the existing BSL does **not** have `resticRepoPrefix`, **do not add one**. The BSL remains Kopia-era.
- **When creating a new BSL:**
  - On Velero **< 1.17**, include `resticRepoPrefix`.
  - On Velero **>= 1.17**, omit `resticRepoPrefix`.

This rule applies to all destination types (AWS, GCP, Azure, S3-compatible, Internal, NFS, HostPath). The presence of `resticRepoPrefix` is the only reliable signal because:

- Velero version alone is not enough: a 1.17+ cluster may still have a Restic-era BSL from before the upgrade.
- Destination type alone is not enough: the same HostPath/NFS/PVC destination can be used with either Restic or Kopia.
- KOTS does not migrate or rewrite backup data; it only updates the BSL config.

Implementation note: `ConfigureStoreOptions` should carry the existing BSL config (if any) **and** the target Velero version. The `resticRepoPrefix` decision is made by combining those two signals: **preserve if present, otherwise omit on 1.17+**.

### 2\. kURL repo (`replicatedhq/kURL`)

A minimal description of the required changes:

- **Add Velero 1.17.x and 1.18.x as add-on versions:** create new directories under `addons/velero/`, update `web/src/installers/versions.js` so the installer config screen offers the new versions, and run the existing `cron-velero-update` generator or manually mirror the 1.16.2 add-on structure.  
- **Update the Velero add-on install template:** in `addons/velero/template/base/install.tmpl.sh`, change the hard-coded `--use-node-agent --uploader-type=restic` to be version-aware (use the same version brackets as KOTS). For Velero 1.17+, use `--use-node-agent` only.  
- **Update the timeout flag mapping:** the template currently maps `resticTimeout` to `--restic-timeout`. For Velero 1.17+, map the same config value to `--fs-backup-timeout` (the Velero 1.17+ equivalent). The user-facing `resticTimeout` field can keep its name for backward compatibility.  
- **Update the config screen text:** the kURL installer UI that renders the Velero version list and the `disableRestic` / `resticTimeout` options should clarify that these control the file-system backup uploader (Restic or Kopia) for the selected Velero version. No new fields are required.  
- **Handle the read-only root filesystem case:** if the kURL installer uses a strict security context, the template should add `emptyDir` volumes for Kopia cache/config directories (`/home/cnb/udmrepo` and `/home/cnb/.cache`) when Velero 1.17+ is selected, because Kopia needs writable pod-local directories.

### 3\. LVP plugin repo (`replicatedhq/local-volume-provider`)

**What the LVP plugin supports:**

The LVP plugin is an object-store plugin for Velero. It supports:

- **Restic-based file-system backups** on Velero 1.16 and earlier.
- **Metadata-only backups** on any Velero version where the BSL provider is one of the LVP providers (`replicated.com/hostpath`, `replicated.com/nfs`, or `replicated.com/pvc`). These operations use the object-store interface, not Kopia.

**What it does not support:**

- **Kopia file-system backups on Velero 1.17+.** This is a fundamental incompatibility, not a bug in the LVP plugin.

**Why Kopia cannot use the LVP plugin:**

Velero 1.17’s Kopia integration does not use the object-store plugin interface. It has its own built-in backends: `s3`, `azure`, `gcs`, and `filesystem`. Velero maps the BSL provider to one of these backends in `pkg/repository/config/config.go` and `pkg/repository/provider/unified_repo.go`. Only these providers are recognized:

```go
// pkg/repository/config/config.go
const (
    AWSBackend   BackendType = "velero.io/aws"
    AzureBackend BackendType = "velero.io/azure"
    GCPBackend   BackendType = "velero.io/gcp"
    FSBackend    BackendType = "velero.io/fs"
)

func IsBackendTypeValid(backendType BackendType) bool {
    return (backendType == AWSBackend || backendType == AzureBackend || backendType == GCPBackend || backendType == FSBackend)
}
```

The LVP plugin registers these providers:

```go
// replicatedhq/local-volume-provider/cmd/local-volume-provider/main.go
RegisterObjectStore("replicated.com/hostpath", newHostPathObjectStorePlugin).
RegisterObjectStore("replicated.com/nfs", newNFSObjectStorePlugin).
RegisterObjectStore("replicated.com/pvc", newPVCObjectStorePlugin).
```

None of these are recognized by Kopia. When Kopia tries to initialize or connect to a repository against an LVP BSL, Velero returns an **"invalid storage provider"** error.

**Why a custom object-store plugin is not enough for Kopia**

The Velero docs you linked are correct that custom plugins can supply object-store backup locations. The LVP plugin does exactly that: it implements Velero’s `ObjectStore` interface and serves backup metadata, logs, and restore logs. That part works fine.

The missing piece is that **Velero’s Kopia repository layer does not use the object-store plugin interface.** Kopia’s repository is handled by a separate, non-pluggable layer in Velero (the Unified Repository / Kopia integration). It has its own built-in backends (`s3`, `azure`, `gcs`, `filesystem`) and chooses the backend by looking up the BSL provider name in a hardcoded map. The LVP plugin’s object-store implementation is never invoked for Kopia repository operations.

So the issue is not “can we write a custom object-store plugin?” — we already did that. The issue is “can Kopia use a custom object-store plugin as its repository backend?” — and the answer in Velero 1.17/1.18 is no. It can only use the four built-in backends, and it only recognizes the four provider names above.

**What about `velero.io/fs`?**

`velero.io/fs` is one of the four recognized Kopia backend types, so it would allow Kopia to use its own filesystem blob storage backend. However, `velero.io/fs` is **not** a built-in Velero object-store plugin. Velero's `pkg/cmd/server/plugin/plugin.go` registers internal item-action plugins, but it does not register an object-store plugin for `velero.io/fs`. If the BSL provider were set to `velero.io/fs`, Velero would fail to load the object-store plugin that handles backup metadata (the tarball, JSON files, logs, etc.).

In other words, `velero.io/fs` alone does not let you use a local filesystem path for full backups. You would still need an object-store plugin that:

1. Registers as `velero.io/fs` (or a provider that Kopia maps to `FSBackend`).
2. Implements the object-store interface on a local filesystem path (mounting the path, serving PutObject/GetObject/ListObjects, etc.).

The LVP plugin could theoretically be extended to register `velero.io/fs` in addition to its existing providers, but that is a code change in the LVP plugin and is not part of this proposal.

**Code changes needed in the LVP repo:**

- **None for Kopia support.** The LVP plugin does not need code changes to support Kopia because the incompatibility is at the Velero/Kopia backend mapping layer.
- **Documentation changes only:** Update the `README.md` and `examples/hostPath.yaml`, `examples/nfs.yaml`, and `examples/pvc.yaml` to state that LVP is not compatible with Kopia file-system backups. Keep the existing Restic examples for Velero 1.16 and earlier. Optionally, recommend S3-compatible storage (e.g., Minio) for local file-system backups on Velero 1.17+.

**What KOTS and kURL should do:**

- For **new** Kopia-era BSLs (Velero >= 1.17), do **not** use LVP providers (`replicated.com/hostpath`, `replicated.com/nfs`, `replicated.com/pvc`) for file-system backups. Use an S3-compatible object store instead. KOTS already uses the S3-compatible Minio internal-storage path (`provider: aws` with `s3Url`) when filesystem Minio is enabled; Kopia maps that to S3 and it works.
- For **existing** LVP BSLs created before the upgrade, preserve them for Restic restores. Warn the customer that the BSL cannot be used for new Kopia file-system backups; a Kopia-compatible BSL (e.g., S3-compatible storage) is required for new FSB backups.
- For **Velero < 1.17**, LVP can still be used with Restic as it is today.

**Read-only root filesystem note:**

Kopia requires writable directories (`/home/cnb/udmrepo`, `/home/cnb/.cache`) regardless of the storage backend. This is not specific to LVP and must be handled in the KOTS/kURL Velero install templates for Kopia. See the KOTS and kURL sections above.

### 4\. Docs repo (`replicated-docs`)

A minimal description of the required changes:

- Update `docs/partials/snapshots/_velero-compatibility.mdx` to list Velero 1.17.x and 1.18.x as supported and against which Kubernetes versions of both KOTS and Kubernetes.  
- Update `docs/enterprise/snapshots-overview.mdx` to replace "Replicated supports only the restic backup program for pod volume data" with a statement that KOTS supports Velero's file-system backup uploader (Restic for Velero 1.16 and earlier, Kopia for Velero 1.17 and later).  
- Update `docs/enterprise/snapshots-storage-destinations.md` to show the version-aware install flags and to remove the hard-coded `--uploader-type=restic` from Velero 1.17+ examples.  
- Update `docs/enterprise/snapshots-velero-cli-installing.md` and `docs/enterprise/snapshots-velero-installing-config.mdx` to mention Kopia for 1.17+ and to update node-agent memory/timeout guidance.  
- Update `docs/enterprise/snapshots-troubleshooting-backup-restore.md` to add a Kopia troubleshooting section (e.g., data mover pods, BackupRepository, read-only root filesystem) and keep the Restic section for older versions.  
- Update kURL Velero add-on docs to list the new add-on versions and clarify the `disableRestic` / `resticTimeout` semantics.  
- Add a section describing the appropriate way to upgrade Velero  
  - one version at a time (kURL manages this but KOTS with existing cluster relies on the cluster admin)  
  - when backups need to be made  
  - when backups made with versions 1.16 and below are abe to be restored up to 1.18.

## High-level release plan

The work can be delivered as four independent, backward-compatible releases. Each release should be tested against both Restic and Kopia, and against the previous versions of the other repos, to ensure no cross-repo breakage.

| \# | Repo | Goal | What changes | Depends on | Backward compatibility | Rollback |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| 1 | `kots` | KOTS emits correct install flags for any Velero version | Version-aware instructions in `pkg/print`, `ConfigureSnapshots.jsx`, and e2e helpers | None | Keeps Restic flags for Velero \< 1.17 | Roll back KOTS binary |
| 2 | `kots` \+ docs | KOTS generates Kopia-compatible BSL config and documents the LVP limitation | Conditional `resticRepoPrefix` in `pkg/snapshot/store.go`; docs update stating LVP is not supported for Kopia FSB; no new LVP BSLs for Kopia | None | Existing Restic BSLs are untouched | Roll back KOTS binary |
| 3 | `kURL` | kURL offers and installs Velero 1.17/1.18 | New Velero add-on versions, version-aware template flags, config screen, timeout flag mapping | None | Older Velero add-on versions remain available | Pin installer to a previous Velero add-on version |
| 4 | `local-volume-provider` | Document Kopia incompatibility | README/examples updates stating LVP is not supported for Kopia FSB; no code changes | None | Restic behavior unchanged | Revert docs |

**Notes on independence:**

- Release 1 (KOTS install flags) can ship first.  
- Release 2 (KOTS BSL config + docs) can ship independently of the LVP docs release.  
- Release 3 (kURL) can ship independently of the LVP docs release.  
- Release 4 (LVP docs) is documentation-only and can ship at any time.  
- Releases 1 and 2 can be combined into a single KOTS release if that is simpler for the team.

## Communication plan

1. **Release notes:** Clearly state that KOTS now supports Velero 1.17 and 1.18 (Kopia) while preserving Restic support for earlier versions. Include a note that customers can upgrade Velero at their own pace and that existing Restic backups are not migrated or deleted.  
2. **Docs:** Publish updated install, storage-destination, and troubleshooting topics before or alongside the release. Include a short migration/upgrade page explaining the version-by-version upgrade path.  
   - Ensure that it is clear that upgrading from 1.16 or earlier means that the existing backups made with Restic are able to be restored until 1.19 is installed.  
   - Explain that new backups are made using Kopia.  
   - State explicitly that Restic restore support will be removed in a future Velero version, so they should not rely on pre-migration backups as their only DR safety net long-term  
3. **In-product messages:** The Admin Console "Install Velero" instructions will automatically show the correct command for the detected Velero version, so no manual customer action is required. No new banner or warning is needed unless we want to proactively tell customers on Restic that they are on a supported path.  
   - Ensure that the message in the instructions includes a warning that a backup should be taken immediately even if there are existing backups, in order to consider the DR functional.  
4. **Support enablement:** Provide a Community article with a short note on Kopia vs Restic repository paths, the `resticRepoPrefix` behavior, the read-only-root-filesystem workaround, and the fact that the LVP plugin is not compatible with Kopia FSB.

## How we will be confident the result works for both Velero 1.16 and 1.17/1.18

| Confidence measure | How we will do it |
| :---- | :---- |
| **Install flag correctness** | Run the e2e smoke test with Velero `v1.16.x` and `v1.18.x`. Verify `velero install` exits 0 in both cases. |
| **AWS S3 snapshots** | Run regression backup/restore with the AWS plugin on 1.16 and 1.18. |
| **LVP HostPath / NFS snapshots with Restic** | Run backup/restore with the LVP plugin on Velero 1.16. Confirm `PodVolumeBackup` and `ResticRepository` objects are created. |
| **Kopia FSB on S3-compatible storage** | Run backup/restore on Velero 1.18 with `provider: aws` and `s3Url`. Confirm `PodVolumeBackup` and `BackupRepository` objects are created. |
| **Kopia read-only root filesystem** | Run Kopia backup/restore on Velero 1.18 with a strict security context and verify that `emptyDir` mounts at `/home/cnb/udmrepo` and `/home/cnb/.cache` allow backups to succeed. |
| **KOTS BSL config for LVP** | Verify that KOTS does not create new LVP-backed BSLs for Velero 1.17+ and that existing LVP BSLs preserve `resticRepoPrefix`. |
| **Backward compatibility** | Create a backup with Velero 1.16, upgrade Velero to 1.17 (without deleting the BSL), and restore the backup. |
| **kURL add-on versions** | Add 1.17.x and 1.18.x to the kURL testgrid and run at least one install smoke test per version. |
| **Airgap image list** | Verify that the e2e image-prep helper pushes all images needed for both versions (velero, AWS plugin, LVP plugin, restore-helper). |
| **Unit tests** | Build `pkg/snapshot` and `pkg/kotsadmsnapshot` with the new version-aware helpers; update log parser tests. |

## Risks and mitigations

| Risk | Mitigation |
| :---- | :---- |
| Removing `resticRepoPrefix` from existing BSLs breaks Restic restores. | Only omit it for new Kopia-era stores; preserve the existing key on Restic-era stores. |
| Kopia data mover pods fail with read-only root filesystem. | Add `emptyDir` volumes for Kopia cache/config when strict security contexts are used. Document the workaround. |
| Customers try to use LVP HostPath/NFS/PVC for Kopia FSB and backups fail. | Document clearly that LVP is not compatible with Kopia FSB; ensure KOTS does not create new LVP-backed BSLs for Velero 1.17+; direct customers to S3-compatible storage for local FSB. |
| kURL install template still emits `--uploader-type=restic` for 1.17+. | Update the template to branch on `VELERO_VERSION`. |
| Customers are confused by “Restic” vs “Kopia” terminology. | Docs and in-product text use generic “file-system backup” language, with version-specific notes. |
| e2e matrix doubles runtime. | Run the full dual-version matrix only on snapshot-related changes; keep the default PR pin on a recent version. |

## References

- Detailed research findings: `proposals/velero_kopia_support_research.md`  
- KOTS snapshot packages: `pkg/snapshot`, `pkg/kotsadmsnapshot`  
- Local-volume-provider plugin (cloned for analysis): `/var/folders/4r/xl9dbxjd7m583dm3ppmk0lqh0000gn/T/opencode/local-volume-provider`  
- kURL repo (cloned for analysis): `/Users/xav/go/src/github.com/replicatedhq/kURL`  
- Docs repo (cloned for analysis): `/Users/xav/work/replicated-docs`  
- Velero 1.17 release notes: [https://github.com/vmware-tanzu/velero/releases/tag/v1.17.0](https://github.com/vmware-tanzu/velero/releases/tag/v1.17.0)  
- Velero 1.17 FSB docs: [https://velero.io/docs/v1.17/file-system-backup/](https://velero.io/docs/v1.17/file-system-backup/)  
- Velero Restic deprecation: [https://velero.io/docs/v1.17/file-system-backup/\#restic-deprecation](https://velero.io/docs/v1.17/file-system-backup/#restic-deprecation)
