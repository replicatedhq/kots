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
- **KOTS does not delete or migrate Restic repositories.** New backups on Velero 1.17+ will use Kopia repositories alongside the existing Restic data.  
- **New backups use Kopia automatically.** The first scheduled or manually triggered backup after the Velero upgrade will create Kopia repositories. There is no need to force a new backup.  
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

KOTS will not automate these steps, but it will surface the correct version-aware instructions and a link to the official Velero upgrade docs. **No BSL reconfiguration is required**: existing Restic backups remain restorable, and new backups will use Kopia automatically.

After the upgrade, if the customer re-configures the storage destination through KOTS (e.g., `kubectl kots velero configure-hostpath`), KOTS will generate a Kopia-compatible BSL config (Release 3).

### New customers

- New installs will use whatever Velero version they provide. KOTS will generate the correct command and storage config for that version.  
- The Admin Console destination picker (AWS, GCP, Azure, S3-compatible, Internal, NFS, HostPath) does not change; it only drives the backend config differently based on the Velero version.
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
| `pkg/snapshot/store.go` | Preserve `resticRepoPrefix` on existing Restic-era LVP/PVC stores; omit it for new Kopia-era stores. See the explicit detection rule below. |
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

The LVP plugin is a filesystem ObjectStore. Kopia can store repository data through the same interface, but Restic-era assumptions in the plugin create operational gaps for Kopia.

#### LVP limitations found in code review

| Limitation | Why it can break Kopia | Why it didn't break Restic |
| :---- | :---- | :---- |
| **Kopia does not recognize LVP provider names** | Velero's Kopia backend mapping only accepts `velero.io/aws`, `velero.io/azure`, `velero.io/gcp`, and `velero.io/fs`. The LVP plugin registers `replicated.com/hostpath`, `replicated.com/nfs`, and `replicated.com/pvc`, which Velero's `GetBackendType` treats as invalid for Kopia. | Restic used the LVP object-store plugin directly via `resticRepoPrefix`; it did not need a Kopia backend mapping. |
| **Read-only root filesystem** | Kopia needs writable `/home/cnb/udmrepo` and `/home/cnb/.cache` in the Velero/node-agent/data mover pods. | Restic did not require writable pod-local directories. |
| **Data mover pods may not inherit node-agent security context** | The LVP plugin sets `fsGroup`/`runAsUser` only on the node-agent DaemonSet; Kopia data mover pods are created separately and may not get the same context. | Restic ran inside the node-agent container itself, so it inherited the context. |
| **`ListObjects` returns directories** | `ListObjects` in `pkg/plugin/plugin.go` returns both files and directories. Kopia's blob layer expects a flat list of objects; receiving directory names like `kopia/ns1/q` can cause it to try to read a directory as an object. | Restic apparently tolerated directory entries in listings. |
| **Multi-node HostPath scheduling** | Kopia data mover pods inherit node-agent volumes; if Velero schedules them on a different node, the HostPath mount points to the wrong filesystem. | Restic ran in-process in the node-agent, so it was always on the same node. |
| **PVC requires RWX** | The plugin creates PVCs with `ReadWriteMany`. | Same limitation for Restic. |
| **No file locking / concurrency support** | `PutObject`/`GetObject`/`DeleteObject` are simple file operations. Multiple Kopia data mover pods could race on repository files. | Restic ran serially in the node-agent. |

#### Kopia backend compatibility with the LVP plugin

This is the most important finding from the Velero 1.17 code review.

**Velero's Kopia repository does not use the object-store plugin interface.** It has its own built-in backends: `s3`, `azure`, `gcs`, and `filesystem`. The BSL provider is mapped to one of these backends in `pkg/repository/config/config.go` and `pkg/repository/provider/unified_repo.go`:

```go
// pkg/repository/config/config.go
const (
    AWSBackend   BackendType = "velero.io/aws"
    AzureBackend BackendType = "velero.io/azure"
    GCPBackend   BackendType = "velero.io/gcp"
    FSBackend    BackendType = "velero.io/fs"
)

func GetBackendType(provider string, config map[string]string) BackendType {
    ...
    if IsBackendTypeValid(bt) {
        return bt
    } else if config != nil && config["s3Url"] != "" {
        return AWSBackend  // S3-compatible stores are treated as AWS
    } else {
        return bt
    }
}

func IsBackendTypeValid(backendType BackendType) bool {
    return (backendType == AWSBackend || backendType == AzureBackend || backendType == GCPBackend || backendType == FSBackend)
}
```

If the BSL provider is not one of those four recognized types and there is no `s3Url`, Kopia returns an **"invalid storage provider"** error when it tries to initialize or connect to the repository.

The LVP plugin registers these providers:

```go
// replicatedhq/local-volume-provider/cmd/local-volume-provider/main.go
RegisterObjectStore("replicated.com/hostpath", newHostPathObjectStorePlugin).
RegisterObjectStore("replicated.com/nfs", newNFSObjectStorePlugin).
RegisterObjectStore("replicated.com/pvc", newPVCObjectStorePlugin).
```

None of these are recognized by Kopia. Therefore, with the current LVP plugin and current Velero 1.17 code, **Kopia cannot create a repository against a BSL that uses `replicated.com/hostpath`, `replicated.com/nfs`, or `replicated.com/pvc`. File-system backups using HostPath/NFS/PVC will fail on Velero 1.17+.**

**What still works:**

- The LVP plugin is still installed as an init container and is still loaded by Velero.
- Backups that do not use file-system backup (i.e., backups that store only Kubernetes metadata and CSI/native volume snapshots) can still use the LVP object-store plugin for metadata storage, because those operations use the object-store interface, not Kopia.

**What does not work:**

- Kopia FSB backups against HostPath/NFS/PVC destinations, because Kopia cannot map the LVP provider to a supported backend.

**What would need to change to make it work:**

There are a few options, listed from least to most invasive:

1. **For KOTS internal storage, keep the S3-compatible Minio path when Kopia is used.** KOTS already uses the Minio path (`provider: aws`, `s3Url: ...`) unless the user has explicitly disabled filesystem Minio. Kopia recognizes `aws`+`s3Url` as S3, so FSB works. This means we should not automatically migrate Minio-backed internal storage to LVP when upgrading to Velero 1.17+.
2. **Make the LVP plugin register a `velero.io/fs` provider** (or alias) and configure the BSL with `provider: velero.io/fs`. Kopia would then map it to `FSBackend` and use Kopia's own filesystem backend to write to the mounted path. The LVP plugin would still be needed to mount the path, but the object-store provider name would change.
3. **Modify Velero's `GetBackendType`** to map `replicated.com/hostpath`, `replicated.com/nfs`, and `replicated.com/pvc` to `FSBackend`. This requires a change to upstream Velero and is outside KOTS's control.
4. **Deprecate LVP-backed HostPath/NFS/PVC for FSB on Velero 1.17+** and direct customers to S3-compatible storage (Minio) for local storage. This is a product decision with support and docs implications.

#### Read-only root filesystem workaround

When a Velero pod (server, node-agent, or Kopia data mover pod) runs with `readOnlyRootFilesystem: true`, Kopia cannot write its repository config or cache because those paths live on the root filesystem. Velero 1.17+ runs as the `cnb` user by default, so the affected paths are:

- `/home/cnb/udmrepo` — Kopia unified repository config directory
- `/home/cnb/.cache` — Kopia cache directory

Kopia fails with an error such as:

```
failed to wait BackupRepository: backup repository is not ready: error to connect to backup repo: error to connect repo with storage: error to connect to repository: unable to write config file: unable to create config directory: mkdir /home/cnb/udmrepo: read-only file system
```

The workaround is to mount writable `emptyDir` volumes at those paths so Kopia writes to ephemeral pod storage instead of the read-only root filesystem:

```yaml
volumeMounts:
  - mountPath: /home/cnb/udmrepo
    name: udmrepo
  - mountPath: /home/cnb/.cache
    name: cache
volumes:
  - emptyDir: {}
    name: udmrepo
  - emptyDir: {}
    name: cache
```

**Where the workaround must be applied:**

| Component | Who can apply the workaround | Notes |
| :---- | :---- | :---- |
| **Velero deployment** | LVP plugin, KOTS, or kURL install template | The Velero server needs `/home/cnb/udmrepo` to initialize the `BackupRepository` CR. |
| **node-agent DaemonSet** | LVP plugin, KOTS, or kURL install template | The node-agent pod needs `/home/cnb/.cache` for Kopia cache. |
| **Kopia data mover pods** | Velero node-agent configmap or data-mover pod template | These pods are created dynamically by Velero; the LVP plugin cannot patch them after creation. They need the same `emptyDir` mounts. |

For KOTS and kURL, the safest approach is to add the `emptyDir` mounts in the install template when Velero 1.17+ is selected. The LVP plugin can also add them as a fallback when it patches the deployment and DaemonSet, but the data-mover pod mounts must come from the node-agent configmap that Velero reads.

#### Backward-compatible changes LVP can make now, independently of KOTS

These changes are safe for existing Restic installations and can be released in the LVP plugin before the KOTS work is ready:

1. **Add runtime Restic/Kopia detection** — `pkg/plugin/kubernetes.go`. Inspect the Velero deployment image tag and node-agent args to decide whether to operate in Restic or Kopia mode. Add a ConfigMap override for clusters with non-standard images.  
2. **Fix `ListObjects` to exclude directories** — `pkg/plugin/plugin.go:182-203`. Filter out `dirEntry.IsDir()` from the returned list. This is more correct for any uploader and should not break Restic.  
3. **Add `"kopia"` to `getSubDirectoryLayout()`** — `pkg/plugin/util.go:28-36`. Pre-create the `kopia` directory alongside `restic`. Safe for Restic.  
4. **Add volume mounts to all node-agent containers, not just `Containers[0]`** — `pkg/plugin/kubernetes.go:208-209`. Safer if the node-agent ever has sidecars.  
5. **Fix `sliceContainsString` to use exact matching** — `pkg/plugin/util.go:38-45`. Avoids accidental substring matches (e.g., matching `lost+found` inside another directory name).  
6. **Fix `DeleteObject` cleanup to recurse beyond two key levels** — `pkg/plugin/plugin.go:221-238`. Kopia keys can be deeper than Restic keys, so empty directories may be left behind.  
7. **Add a ConfigMap option to inject Kopia cache `emptyDir` volumes** — `pkg/plugin/kubernetes.go`. Add an opt-in or detection-based setting (e.g., `kopiaCacheVolumes: true` or `uploader: kopia`) that mounts writable `emptyDir` volumes at the Kopia cache paths for the Velero deployment and node-agent DaemonSet. This is independent of KOTS and can be consumed by kURL or KOTS later.  
8. **Preserve Kopia cache volume names in `preserveVolumes` cleanup** — `pkg/plugin/kubernetes.go:227-256`. If Kopia cache volumes are added, make sure `removeUnusedVolumes` does not strip them.

#### Can a single LVP release support both Restic and Kopia?

**Yes.** The LVP plugin is a single Go binary that can detect the active uploader at runtime and branch its behavior. This means one LVP release can support both Restic and Kopia clusters without requiring KOTS to release in lockstep.

**How LVP can detect which uploader is active:**

| Detection method | Where it comes from | Reliability |
| :---- | :---- | :---- |
| **Velero version from deployment image tag** | The LVP plugin already reads the Velero deployment to patch it; it can parse the image tag (e.g., `velero/velero:v1.17.1`). | High for standard images. Fragile for custom/private images with non-standard tags. |
| **Node-agent DaemonSet args** | The plugin already reads the node-agent DaemonSet; it can look for `--uploader-type=restic` or `--uploader-type=kopia`. | High for Velero 1.10–1.16. For 1.17+ the flag is usually omitted because Kopia is the default. |
| **BSL config `resticRepoPrefix`** | The BSL config is passed to `Init()`. If the key is present and the Velero version is \< 1.17, Restic is likely in use. | Low as a standalone signal, because KOTS may still send `resticRepoPrefix` for Kopia-era BSLs. |
| **Existing repository CRs** | The plugin can list `BackupRepository` (Kopia) vs `ResticRepository` (Restic) CRs in the Velero namespace. | Useful for transition clusters, but not available before the first backup. |

**Recommended detection strategy:**

1. Parse the Velero deployment image tag.  
   - `< 1.10` → Restic.  
   - `1.10–1.16` → check node-agent args for `--uploader-type`; default to Restic if absent.  
   - `>= 1.17` → Kopia.  
2. Allow a ConfigMap override (e.g., `uploader: restic` or `uploader: kopia`) so vendors can force a mode if image-tag parsing fails.  
3. If detection is ambiguous, default to the current Restic-safe behavior.

**What LVP can do differently based on detection:**

- **Kopia mode:** add `kopia` cache `emptyDir` volumes, pre-create the `kopia` subdirectory, and ensure `ListObjects` filters directories.  
- **Restic mode:** keep current behavior, including `restic` subdirectory creation and no Kopia cache mounts.  
- **Both modes:** the `ListObjects` directory fix and `sliceContainsString` exact-match fix are safe to apply universally.

**Why this is valuable:**

- KOTS and kURL can consume the same LVP image regardless of the Velero version they are paired with.  
- The LVP plugin can be released on its own schedule, which aligns with the decision to tag the LVP plugin with doc changes.  
- It reduces the risk that a KOTS upgrade inadvertently breaks an older Restic cluster by pulling in a Kopia-only plugin.

#### LVP changes that require coordination with KOTS/kURL

- Making the Kopia cache `emptyDir` mounts **default-on** rather than opt-in or detection-based, because the cache path depends on the Velero image user (default `cnb` for Velero 1.17+, but custom images may differ).  
- Handling read-only root filesystem in the kURL/KOTS install templates themselves (the LVP plugin can only patch what the templates give it).  
- Deciding whether the LVP plugin should also detect the Velero version from the deployment image tag to auto-enable Kopia cache mounts, or rely on the ConfigMap override.

#### Documentation and release changes

- Tag a new plugin release (e.g. `v0.6.0`) with the code and README updates.  
- Update `README.md` and `examples/hostPath.yaml`, `examples/nfs.yaml`, `examples/pvc.yaml` to show Kopia-compatible install commands and to remove `resticRepoPrefix` from default examples, while keeping it in legacy Restic examples.

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
| 1 | `local-volume-provider` | LVP supports both Restic and Kopia in one image | Runtime uploader detection, `ListObjects` directory fix, `kopia` subdir, opt-in Kopia cache `emptyDir` volumes, docs/examples | None | Restic behavior is unchanged; Kopia features are opt-in or detection-based | Revert to previous LVP image tag |
| 2 | `kots` | KOTS emits correct install flags for any Velero version | Version-aware instructions in `pkg/print`, `ConfigureSnapshots.jsx`, and e2e helpers | None | Keeps Restic flags for Velero \< 1.17 | Roll back KOTS binary |
| 3 | `kots` \+ docs | KOTS generates Kopia-compatible LVP/PVC BSL config | Conditional `resticRepoPrefix` in `pkg/snapshot/store.go`, updated docs | Release 1 (Kopia LVP stores need the new plugin) | Existing Restic BSLs are untouched | Roll back KOTS binary |
| 4 | `kURL` | kURL offers and installs Velero 1.17/1.18 | New Velero add-on versions, version-aware template flags, config screen, timeout flag mapping | Release 1 (new add-on versions reference the new LVP image) | Older Velero add-on versions remain available | Pin installer to a previous Velero add-on version |

**Notes on independence:**

- Release 1 (LVP) can ship first and is safe for existing Restic clusters on its own.  
- Release 2 (KOTS install flags) can ship before or after Release 1; it only changes the commands KOTS shows to users.  
- Release 3 (KOTS BSL config) should ship after Release 1 so Kopia-backed LVP/PVC stores actually work end-to-end.  
- Release 4 (kURL) should ship after Release 1 so the new Velero add-on versions reference a Kopia-capable LVP image.  
- Releases 2 and 3 can be combined into a single KOTS release if that is simpler for the team.

## Communication plan

1. **Release notes:** Clearly state that KOTS now supports Velero 1.17 and 1.18 (Kopia) while preserving Restic support for earlier versions. Include a note that customers can upgrade Velero at their own pace and that existing Restic backups are not migrated or deleted.  
2. **Docs:** Publish updated install, storage-destination, and troubleshooting topics before or alongside the release. Include a short migration/upgrade page explaining the version-by-version upgrade path.  
   - Ensure that it is clear that upgrading from 1.16 or earlier means that the existing backups made with Restic are able to be restored until 1.19 is installed.  
   - Explain that new backups are made using Kopia.  
   - State explicitly that Restic restore support will be removed in a future Velero version, so they should not rely on pre-migration backups as their only DR safety net long-term  
3. **In-product messages:** The Admin Console "Install Velero" instructions will automatically show the correct command for the detected Velero version, so no manual customer action is required. No new banner or warning is needed unless we want to proactively tell customers on Restic that they are on a supported path.  
   - Ensure that the message in the instructions includes a warning that a backup should be taken immediately even if there are existing backups, in order to consider the DR functional.  
4. **Support enablement:** Provide a Community article with a short note on Kopia vs Restic repository paths, the `resticRepoPrefix` behavior, and the read-only-root-filesystem workaround.

## How we will be confident the result works for both Velero 1.16 and 1.17/1.18

| Confidence measure | How we will do it |
| :---- | :---- |
| **Install flag correctness** | Run the e2e smoke test with Velero `v1.16.x` and `v1.18.x`. Verify `velero install` exits 0 in both cases. |
| **AWS S3 snapshots** | Run regression backup/restore with the AWS plugin on 1.16 and 1.18. |
| **LVP HostPath / NFS snapshots** | Run backup/restore with the LVP plugin on 1.16 and 1.18. Confirm `PodVolumeBackup` and `BackupRepository` (or `ResticRepository`) objects are created. |
| **LVP data mover pods** | On Velero 1.17+, verify that Kopia data mover pods are spawned and inherit the LVP volume from the node-agent DaemonSet. |
| **LVP plugin fixes** | After applying the LVP `ListObjects` fix and Kopia cache `emptyDir` option, run HostPath/NFS/PVC backup/restore on Velero 1.18 and confirm the prior failure is resolved. |
| **LVP read-only root filesystem** | Run LVP HostPath with a strict security context and verify that Kopia cache `emptyDir` mounts allow backups to succeed. |
| **Backward compatibility** | Create a backup with Velero 1.16, upgrade Velero to 1.17 (without deleting the BSL), and restore the backup. |
| **kURL add-on versions** | Add 1.17.x and 1.18.x to the kURL testgrid and run at least one install smoke test per version. |
| **Airgap image list** | Verify that the e2e image-prep helper pushes all images needed for both versions (velero, AWS plugin, LVP plugin, restore-helper). |
| **Unit tests** | Build `pkg/snapshot` and `pkg/kotsadmsnapshot` with the new version-aware helpers; update log parser tests. |

## Risks and mitigations

| Risk | Mitigation |
| :---- | :---- |
| Removing `resticRepoPrefix` from existing BSLs breaks Restic restores. | Only omit it for new Kopia-era stores; preserve the existing key on Restic-era stores. |
| Kopia data mover pods fail with read-only root filesystem. | Add `emptyDir` volumes for Kopia cache/config when strict security contexts are used. Document the workaround. |
| LVP `ListObjects` returns directories, confusing Kopia's blob layer. | Fix `ListObjects` in the LVP plugin to filter out directories; this is backward-compatible and can ship before KOTS. |
| Kopia data mover pods cannot access the LVP volume due to security context or node scheduling. | Test with the LVP plugin fixes in place; use NFS/PVC to avoid multi-node HostPath issues. |
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
