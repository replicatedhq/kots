# Velero Kopia Support – Research Findings

## Goal

Understand how KOTS integrates with Velero for the snapshot/backup feature, identify every place that assumes the legacy Restic-based file-system backup (FSB) path, and document how a move to Velero’s newer Kopia-based FSB path would affect the codebase.

## TL;DR

* The KOTS repo is currently hard-coded to the **Restic** path: install instructions, e2e setup, and the local-volume-provider filesystem store all use `--uploader-type=restic` / `resticRepoPrefix`.
* Velero **v1.17 removed `--uploader-type=restic`** and made **Kopia** the default/only FSB uploader. Restic backups can still be restored in v1.17/v1.18, but v1.19 will remove even that.
* Updating KOTS to support current Velero versions is therefore not a simple version bump: the install command, the filesystem/LVP storage configuration, repository reset logic, and the e2e test harness all need changes.
* The **Replicated local-volume-provider plugin** is currently documented as Restic-only. It is **not compatible with Kopia** because Velero 1.17+ does not use object-store plugins for Kopia repository operations; Kopia uses its own built-in backends (`s3`, `azure`, `gcs`, `filesystem`). The plugin README/examples should be updated to state this incompatibility and recommend S3-compatible storage (e.g., Minio) for local file-system backups on Velero 1.17+.

## Current Velero versions in the repo

| Location | Version | Notes |
|----------|---------|-------|
| `go.mod` | `github.com/vmware-tanzu/velero v1.18.0` | The Go types/API client the KOTS binary is compiled against. |
| `e2e/scripts/deps.sh` | `v1.16.2` (pinned) | Comment explicitly says: *“pin to 1.16.2 as 1.17.0 removed restic support”*. Tests run against an older CLI to keep the existing Restic assertions working. |

This means KOTS is already compiling against Velero 1.18 APIs but is not exercising the CLI behaviour of 1.17+ in CI.

## Where Velero is used

### Core snapshot packages

| Package | Purpose |
|---------|---------|
| `pkg/snapshot` | Velero namespace/BSL discovery, store configuration (AWS/S3/Azure/GCP/LVP/Minio), filesystem LVP deployment, backup/restore list, wait helpers. |
| `pkg/kotsadmsnapshot` | Creates KOTS **application** and **instance** backups, parses backup status/volume summary, restores applications, downloads/logs parsing. |
| `pkg/print` | CLI installation instructions for Velero. |
| `cmd/kots/cli` | `kubectl kots velero ...` subcommands. |
| `pkg/kotsadm/objects` | KOTS deployment pod annotations for Velero backup hooks (`backup.velero.io/backup-volumes`, `pre.hook.backup.velero.io/command`). |
| `pkg/supportbundle` | Collects Velero namespace/logs in support bundles. |
| `pkg/store` | Stores snapshot TTL/schedule preferences in the KOTS DB. |

### Web UI

| Component | Purpose |
|-----------|---------|
| `web/src/components/snapshots/ConfigureSnapshots.jsx` | Shows the “Install Velero” modal and the exact flags to pass. |
| `web/src/components/snapshots/SnapshotInstallationBox.jsx` | Checks Velero status and displays the “Node Agent/Restic” component name. |
| `web/src/components/snapshots/SnapshotDetails.jsx` | Displays volume summary from `PodVolumeBackup` resources. |

### E2E tests

| File | Purpose |
|------|---------|
| `e2e/velero/cli.go` | Ginkgo helper that runs `velero install ...` for smoke tests. |
| `e2e/scripts/deps.sh` | Downloads the pinned Velero CLI. |
| `e2e/playwright/regression/shared/cli.ts` | Playwright regression helpers: `installVeleroAWS`, `installVeleroHostPath`, `prepareVeleroImages`. |

## Restic-specific assumptions found in the code

### 1. Velero install flags hard-coded to `restic`

These files generate the exact `velero install` command shown to users or used in tests:

| File | Line(s) | Current value | Problem with Velero ≥ 1.17 |
|------|---------|---------------|---------------------------|
| `pkg/print/velero.go` | 21-27, 44-50, 81, 107 | `--use-node-agent --uploader-type=restic` | `--uploader-type=restic` is rejected. Kopia is default. |
| `web/src/components/snapshots/ConfigureSnapshots.jsx` | 46-49 | `isVelero10OrNewer ? ["--use-node-agent", "--uploader-type=restic"] : ["--use-restic"]` | Same as above. The version gate is still useful, but the uploader type must change. |
| `e2e/playwright/regression/shared/cli.ts` | 88, 126 | `--use-node-agent --uploader-type=restic` / `--use-restic` | Used by regression tests. |
| `e2e/velero/cli.go` | 60-61 | `--use-node-agent --uploader-type=restic` | Used by smoke tests. |

The pre-1.10 branch (`--use-restic`) is only relevant for very old Velero versions; for any supported modern Velero the command should be `--use-node-agent` (and optionally `--uploader-type=kopia`, or omit it because Kopia is the default from 1.17 onward).

### 2. `resticRepoPrefix` in BackupStorageLocation config for LVP/PVC stores

The KOTS code builds the `BackupStorageLocation` config for filesystem/LVP and PVC stores with a hard-coded `resticRepoPrefix`:

| File | Line(s) | Value |
|------|---------|-------|
| `pkg/snapshot/store.go` | 530 | `"resticRepoPrefix": "/var/velero-local-volume-provider/velero-internal-snapshots/restic"` (PVC internal store) |
| `pkg/snapshot/store.go` | 846 | `"resticRepoPrefix": fmt.Sprintf("/var/velero-local-volume-provider/%s/restic", resticDir)` (HostPath LVP) |
| `pkg/snapshot/store.go` | 856 | `"resticRepoPrefix": fmt.Sprintf("/var/velero-local-volume-provider/%s/restic", resticDir)` (NFS LVP) |

Velero’s own documentation explicitly states:

> **Note:** `resticRepoPrefix` doesn’t work for Kopia.

Kopia auto-generates the repository path from the BackupStorageLocation (`…/kopia/<namespace>`). For object storage this is fine, but for the Replicated local-volume-provider plugin the path is currently passed via `resticRepoPrefix`. This is the biggest unknown: the LVP plugin may need to be updated to support Kopia repositories, or KOTS may need to stop using `resticRepoPrefix` and configure the LVP bucket/prefix differently.

There is also an unused constant in the CLI code:

| File | Line | Constant |
|------|------|----------|
| `cmd/kots/cli/velero.go` | 33 | `resticRepoBase = "/var/velero-local-volume-provider"` (not referenced anywhere) |

### 3. Repository reset logic handles both Restic and Kopia repositories

`pkg/snapshot/store.go` `resetRepositories()` (lines 1609-1618) deletes both:

* `BackupRepository` CRs (Velero 1.10+ / Kopia path) – `resetBackupRepositories()` (lines 1621-1647)
* `ResticRepository` CRs (legacy Restic path) – `resetResticRepositories()` (lines 1649-1677)

For Kopia-only operation the Restic reset is harmless but unnecessary. It can be kept for backward compatibility with existing Restic backups, but should be clearly gated or deprecated.

### 4. DaemonSet detection still falls back to the old `restic` name

`pkg/snapshot/velero.go` line 411 searches for node-agent pods with:

```go
nameReq, err := labels.NewRequirement("name", selection.In, []string{"node-agent", "restic"})
```

and line 709-713 checks for the `node-agent` DaemonSet, then falls back to the old `restic` DaemonSet name. This is a backward-compatibility feature; it still works because the label selector also accepts `node-agent`. The fallback can remain for old clusters.

### 5. Log parser tests use Restic-specific log lines

`pkg/kotsadmsnapshot/logparser.go` itself is generic (it parses logfmt output), but the test fixture file `pkg/kotsadmsnapshot/logparser_test.go` contains hard-coded Restic log lines such as:

> `Skipping snapshot of persistent volume because volume is being backed up with restic.`
> `pkg/restic/backupper.go:156`

The parser will continue to work with Kopia logs, but the tests should be updated to reflect real Kopia/Velero 1.17+ log output.

### 6. E2E image preparation references the legacy Restic restore helper

`e2e/playwright/regression/shared/cli.ts` lines 206-230 already branch on `isVelero10OrNewer`:

* `velero-restore-helper` for ≥ 1.10 (still correct for Kopia)
* `velero-restic-restore-helper` for < 1.10 (only correct for old Restic)
* ConfigMap name `fs-restore-action-config` for ≥ 1.10 (still correct)
* ConfigMap label `velero.io/pod-volume-restore: RestoreItemAction` for ≥ 1.10 (still correct)

So the restore-helper image logic is **already fine** for a Kopia move; only the install command needs to change.

### 7. Local-volume-provider plugin is documented as Restic-only and is incompatible with Kopia

The Replicated `local-volume-provider` plugin (used for HostPath/NFS/PVC snapshot stores) is documented as Restic-only:

> *“It also supports volume snapshots with Restic.”*
> *“Must be provided if you're using Restic; [default mount] + [bucket] + [prefix] + 'restic'”*

In practice, the plugin is a generic filesystem-backed ObjectStore plugin that implements the Velero ObjectStore interface. However, **Velero 1.17+ does not use ObjectStore plugins for Kopia repositories** — Kopia is invoked directly by Velero and reads/writes its repository through its own backends (`s3`, `azure`, `gcs`, `filesystem`). Because the plugin's `replicated.com/hostpath`, `replicated.com/nfs`, and `replicated.com/pvc` providers are not valid Kopia backends, the plugin cannot be used for Kopia file-system backups.

For local file-system backups on Velero 1.17+, customers should use an S3-compatible object store such as **Minio** (e.g., provider `aws` with `s3Url` pointing to the local Minio instance). The plugin README and example manifests should be updated to state this incompatibility and migration path.

### 8. Velero documentation links point to v1.10

`pkg/print/velero.go` links users to `https://velero.io/docs/v1.10/...` and uses the v1.10 node-agent docs. For a Kopia-based flow these links should point to the v1.17/v1.18 file-system-backup documentation.

## What Velero/Kopia changes

Based on Velero 1.17 release notes and docs:

| Area | Restic path (old) | Kopia path (new, Velero 1.17+) |
|------|-------------------|-------------------------------|
| Install flag | `--use-restic` (pre-1.10) or `--use-node-agent --uploader-type=restic` (1.10-1.16) | `--use-node-agent` (Kopia is default; optional `--uploader-type=kopia`) |
| Valid `--uploader-type` values | `restic` (removed in 1.17) | `kopia` |
| Repository type | `ResticRepository` CR | `BackupRepository` CR backed by Kopia |
| Repo path hint in BSL | `resticRepoPrefix` | Not used by Kopia; path auto-derived from BSL |
| Restore helper image | `velero-restic-restore-helper` (<1.10) / `velero-restore-helper` (≥1.10) | `velero-restore-helper` |
| Restore helper config map | `restic-restore-action-config` (<1.10) / `fs-restore-action-config` (≥1.10) | `fs-restore-action-config` |
| Volume backup pods | Node-agent runs restic directly | Node-agent spawns **data mover pods** that run Kopia modules |
| Architecture | Monolithic in node-agent | Micro-service; supports cancel/resume/concurrency |

Deprecation timeline from Velero:

* **1.15 / 1.16**: Restic backups still succeed but emit warnings.
* **1.17 / 1.18**: Restic **backups are disabled**; restores from existing Restic backups still work.
* **1.19+**: Restic **restores are also disabled**.

## Local-volume-provider plugin deep dive

I cloned `replicatedhq/local-volume-provider` into `/var/folders/4r/xl9dbxjd7m583dm3ppmk0lqh0000gn/T/opencode/local-volume-provider` to inspect the actual plugin behavior.

### What the plugin does

`cmd/local-volume-provider/main.go` registers three Velero ObjectStore providers:

- `replicated.com/hostpath`
- `replicated.com/nfs`
- `replicated.com/pvc`

`pkg/plugin/plugin.go` implements `LocalVolumeObjectStore`, a simple filesystem-backed object store that stores files under `/var/velero-local-volume-provider/<bucket>/<key>`. During `Init()` it calls `ensureResources()` in `pkg/plugin/kubernetes.go`, which:

1. Mounts the requested volume into the Velero deployment.
2. Mounts the same volume into the node-agent DaemonSet.
3. Adds a `local-volume-provider` fileserver sidecar to the Velero deployment for signed URLs.

### Where the plugin references Restic

| Reference | Location | Notes |
|-----------|----------|-------|
| `ResticDaemonsetName = "restic"` | `pkg/plugin/kubernetes.go:35` | Backward-compatibility fallback only. |
| `name: "restic"` in tests | `pkg/plugin/kubernetes_test.go` | Tests for the old daemonset name. |
| `resticRepoPrefix` | `README.md`, `examples/hostPath.yaml`, `examples/nfs.yaml`, `examples/pvc.yaml` | **Not used in the Go code.** The plugin never reads this key. |
| `--uploader-type=restic` | `README.md` | Install instructions only. |
| `getSubDirectoryLayout()` includes `"restic"` | `pkg/plugin/util.go:28-36` | Pre-creates a `restic` directory; Velero's layout also includes `kopia`, but Kopia will create it on demand. |

### Why the plugin cannot work with Kopia

Although the plugin is **storage-format agnostic** and implements the Velero ObjectStore interface (`Init`, `PutObject`, `GetObject`, `ListObjects`, `DeleteObject`, `ObjectExists`, `CreateSignedURL`), Velero 1.17+ does **not** route Kopia repository data through the ObjectStore plugin layer. Instead, Velero's `BackupRepository` controller invokes Kopia directly, and Kopia uses its own backends (`s3`, `azure`, `gcs`, `filesystem`) to manage the repository. The `replicated.com/hostpath`, `replicated.com/nfs`, and `replicated.com/pvc` providers are not valid Kopia backends, so a BackupStorageLocation using any of those providers cannot be used for Kopia file-system backups.

This has been confirmed by the customer-facing symptom: LVP-based BackupStorageLocations become `Unavailable` on Velero 1.17+ with errors such as `Backup store contains invalid top-level directories`. The plugin's volume mounts into the node-agent DaemonSet are also irrelevant because the Kopia data mover pods do not call the ObjectStore plugin.

### What needs to change in the plugin

1. **README and examples**: explicitly state that the plugin is **not compatible with Kopia** and is only supported on Velero 1.16 or lower (with Restic). Update the compatibility table and install commands to reflect this.
2. **Example manifests**: add a header comment to `examples/hostPath.yaml`, `examples/nfs.yaml`, and `examples/pvc.yaml` warning about the Kopia incompatibility and pointing customers to S3-compatible storage (e.g., Minio) for Velero 1.17+.
3. **No Go API changes are required**: the plugin code itself is not broken; it is simply incompatible with the Kopia path by design.

### What needs to change in KOTS for local file-system backups on Velero 1.17+

- For HostPath/NFS/PVC stores on Velero 1.17+, KOTS should switch to the **filesystem Minio** path (`provider: aws` with `s3Url` pointing to the local KOTS-managed Minio instance) instead of using the LVP plugin.
- The existing `pkg/snapshot/store.go` `ConfigureStore` path may already create an LVP BackupStorageLocation on non-kURL clusters when `minio-enabled-snapshots` is false. This must be gating or redirected to filesystem Minio for Velero 1.17+.
- Remove the unused `resticRepoBase` constant from `cmd/kots/cli/velero.go`.
- Update install instructions to use `--use-node-agent` (without `--uploader-type=restic`).

### LVP/Kopia validation status

- The incompatibility is now considered **confirmed** by Velero/Kopia architecture and customer reports. The PR documenting this is at https://github.com/replicatedhq/local-volume-provider/pull/121.
- End-to-end Kopia validation for LVP is no longer needed because the plugin cannot be used with Kopia; validation effort should shift to the filesystem Minio path.
- Kopia data mover pods require writable cache/config directories. If KOTS sets `ReadOnlyRootFilesystem` on Velero/node-agent pods, emptyDir volumes may need to be added.

## Impact assessment

| Component | Impact | Notes |
|-----------|--------|-------|
| CLI install instructions (`pkg/print/velero.go`) | **High** | Must remove `restic`, update docs links, possibly mention Kopia default. |
| Web install instructions (`ConfigureSnapshots.jsx`) | **High** | Same flag change. |
| Velero configure commands (`cmd/kots/cli/velero.go`) | **Medium** | Only the unused `resticRepoBase` constant; actual storage config is in `pkg/snapshot/store.go`. |
| BackupStorageLocation construction (`pkg/snapshot/store.go`) | **High** | `resticRepoPrefix` must be removed/changed for Kopia. The LVP plugin itself is likely compatible; the main issue is KOTS sending the Restic-only config key. |
| Repository reset (`pkg/snapshot/store.go`) | **Low** | Restic reset can stay as legacy fallback but should be gated. |
| Node-agent detection (`pkg/snapshot/velero.go`) | **Low** | Already supports both names; no change required. |
| Backup/restore creation (`pkg/snapshot`, `pkg/kotsadmsnapshot`) | **Low** | Uses Velero CRs; Kopia is transparent at this layer. |
| Volume summary (`pkg/kotsadmsnapshot/backup.go`) | **Low** | `PodVolumeBackup` resources still exist; Kopia populates them. |
| E2E smoke tests (`e2e/velero/cli.go`) | **High** | Install command must change. |
| E2E regression tests (`e2e/playwright/regression/shared/cli.ts`) | **High** | Install command must change. |
| E2E dependency script (`e2e/scripts/deps.sh`) | **High** | Pin must be bumped; the comment about Restic removal is the reason for the pin. |
| Log parser tests | **Low** | Update fixtures, not runtime logic. |
| KOTS deployment annotations | **None** | Generic backup hooks. |
| Airgap image preparation | **Medium** | Verify `velero-restore-helper` and plugin images are still the only extra images needed. |
| Local-volume-provider plugin (external) | **Medium / Validation needed** | Code is likely Kopia-compatible; needs README/example updates and end-to-end testing. |

## Affected files (actionable list)

| File | What to change |
|------|----------------|
| `pkg/print/velero.go` | Remove `--uploader-type=restic`; replace with `--uploader-type=kopia` or omit it; update doc URLs to v1.17+/latest. |
| `web/src/components/snapshots/ConfigureSnapshots.jsx` | Change `getFSBackupComponentFlags()` to return `["--use-node-agent"]` (or add `--uploader-type=kopia`). Update labels if desired. |
| `e2e/velero/cli.go` | Remove `--uploader-type=restic`. |
| `e2e/playwright/regression/shared/cli.ts` | Remove `--uploader-type=restic` in both `installVeleroAWS` and `installVeleroHostPath`. |
| `e2e/scripts/deps.sh` | Bump `VELERO_RELEASE` to a Kopia-based version (e.g. `v1.17.x` or `v1.18.x`) and remove/update the Restic comment. |
| `pkg/snapshot/store.go` | Decide how to handle `resticRepoPrefix` for LVP/PVC stores under Kopia. Remove or replace the key. Keep Restic reset as legacy fallback. |
| `cmd/kots/cli/velero.go` | Remove unused `resticRepoBase` constant. |
| `pkg/kotsadmsnapshot/logparser_test.go` | Update test fixtures to Kopia/Velero 1.17+ log lines. |
| `go.mod` | Already on `v1.18.0`, so no bump required; but ensure no deprecated API usage. |
| External: `replicatedhq/local-volume-provider` | Update README/examples to state that the plugin is **not compatible with Kopia** and is only supported on Velero 1.16 or lower with Restic. Point customers to S3-compatible storage such as Minio for Velero 1.17+. No Go code changes required. |

## Risks and open questions

1. **LVP plugin support** – The local-volume-provider plugin is **not compatible with Kopia** because Velero 1.17+ does not use ObjectStore plugins for Kopia repositories. LVP-based stores must be migrated to an S3-compatible destination (e.g., Minio) before upgrading to Velero 1.17+. The plugin README/examples have been updated to document this in PR https://github.com/replicatedhq/local-volume-provider/pull/121.
2. **Backward compatibility** – Existing customer Restic backups must still be restorable. Velero 1.17/1.18 allows restores from Restic; KOTS should not force customers to delete old Restic repositories. The `resetResticRepositories` path may be needed for older stores.
3. **Airgap image list** – Velero 1.17+ uses data mover pods for Kopia. In airgap scenarios all required images (velero, plugin, restore-helper, and potentially LVP plugin) must be pre-pushed. The current e2e helper already pushes `velero-restore-helper`; verify no additional image is required.
4. **Node-agent config maps** – Velero 1.17+ introduces a `node-agent` ConfigMap for concurrency/priority/queue settings. KOTS may need to document or configure this for large clusters, but it is not a hard requirement.
5. **Read-only root filesystem** – Kopia needs writable cache/config directories. If KOTS/security policies set `ReadOnlyRootFilesystem`, the node-agent/velero pods must mount emptyDir volumes for Kopia cache. This may affect KOTS-generated Velero deployments.
6. **Version gating** – The UI and CLI currently gate on `isVelero10OrNewer`. For Kopia we likely need a new gate (`isVelero17OrNewer`) to decide whether to emit `--uploader-type=kopia` vs. `--uploader-type=restic` for older versions, or to simply omit the flag for new versions.

## Recommended next steps

1. **Update LVP plugin documentation** – Done in PR https://github.com/replicatedhq/local-volume-provider/pull/121. The README and examples now state that LVP is not compatible with Kopia and recommend S3-compatible storage (e.g., Minio) for Velero 1.17+.
2. **Create a feature branch** and update the install flags in `pkg/print/velero.go`, `ConfigureSnapshots.jsx`, and the e2e helpers to use `--use-node-agent` without Restic.
3. **Update `pkg/snapshot/store.go`** to redirect HostPath/NFS/PVC stores on Velero 1.17+ to filesystem Minio (S3-compatible) instead of the LVP plugin. Stop injecting `resticRepoPrefix` for new Kopia-backed BackupStorageLocations; preserve it only for legacy Restic restores if needed.
4. **Bump the e2e Velero pin** to `v1.17.x` or `v1.18.x` and run the snapshot regression tests.
5. **Update test fixtures** and doc links to match the new Velero version.
6. **Add a version-gate helper** (e.g. `isVelero17OrNewer`) so the UI/CLI can still support older Restic-based installs during a transition window.

## References

* Velero 1.17 release notes: https://github.com/vmware-tanzu/velero/releases/tag/v1.17.0
* Velero 1.17 file-system backup docs: https://velero.io/docs/v1.17/file-system-backup/
* Velero Restic deprecation docs (v1.17): https://velero.io/docs/v1.17/file-system-backup/#restic-deprecation
* Replicated local-volume-provider README: https://github.com/replicatedhq/local-volume-provider
* LVP documentation PR (Kopia incompatibility): https://github.com/replicatedhq/local-volume-provider/pull/121
* Cloned local-volume-provider repo for this analysis: `/var/folders/4r/xl9dbxjd7m583dm3ppmk0lqh0000gn/T/opencode/local-volume-provider`
* Velero data mover pod creation (inherits node-agent volumes): `velero/pkg/exposer/pod_volume.go`
* KOTS snapshot packages: `pkg/snapshot`, `pkg/kotsadmsnapshot`
