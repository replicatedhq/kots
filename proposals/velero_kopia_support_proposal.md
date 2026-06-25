# Proposal: Velero Kopia Support (with Restic preserved)

## Executive Summary

**Problem.** KOTS supports Velero only up to version 1.16, which is tested only through Kubernetes 1.33. Replicated drops Kubernetes 1.33 support at the end of June 2026, so customers on Kubernetes 1.34+ will have no supported Velero option. In addition, Velero 1.17 made Kopia the default file-system backup uploader and removed the `--uploader-type=restic` flag. KOTS still hard-codes Restic in install instructions, tests, and generated configs, so customers on Velero 1.17+ get invalid install commands and failing backups.

**Proposal.** Make KOTS version-aware so it emits the correct Velero install flags and `BackupStorageLocation` configuration for Velero 1.10 through 1.18. Restic support is preserved; customers can upgrade Velero at their own pace and existing Restic backups remain restorable.

**Value.** Unblocks Velero 1.17/1.18 (and Kubernetes 1.34+) support, prevents silent backup failures, and protects existing restore investments without forcing a backup migration.

## Scope

- **Velero versions:** 1.10.1 through **1.18.x** (the currently supported range). Velero **1.19 is out of scope** for now because it removes the ability to restore Restic backups.  
- **KOTS, kURL, and docs** are all in scope because each touches the install experience or compatibility claims.  
- **The LVP plugin is out of scope for Kopia support.** Velero 1.17+ does not use object-store plugins for Kopia repository operations, so the LVP plugin cannot be made Kopia-compatible. KOTS will instead use the internal S3-compatible filesystem Minio for HostPath/NFS/PVC destinations on Velero 1.17+.  
- **No migration** of existing Restic backups.  
- We do not enforce Velero and Kubernetes version coordination, it remains possible to install unsupported combinations.

## Customer problem and why this matters

- **Backup failure after a Velero upgrade.** A customer on Velero 1.16 who upgrades to 1.17+ will receive KOTS-generated install instructions that no longer work (`--uploader-type=restic` is rejected). If they proceed, snapshots fail with no clear warning in the Admin Console.
- **No supported path for Kubernetes 1.34+.** Velero 1.16 is only tested through Kubernetes 1.33. Replicated drops Kubernetes 1.33 support in June 2026, so customers on newer Kubernetes need Velero 1.17/1.18, which are currently unavailable in kURL and unsupported in KOTS.
- **Lost confidence in the platform.** KOTS appears to lag behind a supported upstream dependency, blocking security and bug-fix upgrades for customers who rely on snapshots.

## Customer experience

### Customers who upgrade KOTS only

No action is required. KOTS detects the installed Velero version and generates the correct install instructions for that version. Restic fallback code and documentation are preserved.

### Customers who upgrade Velero to 1.17+

- **Existing Restic backups remain restorable.** Velero 1.17/1.18 still restores from Restic repositories, and KOTS preserves `resticRepoPrefix` on existing BSLs.
- **New backups use Kopia automatically.** The first scheduled or manual backup after the upgrade creates a Kopia repository; no forced migration is needed.
- **BSL compatibility depends on the existing storage type.**

| Existing BSL type | Reconfiguration needed for Kopia FSB? | Notes |
| :---- | :---- | :---- |
| **Minio/S3-compatible** (`provider: aws` with `s3Url`) | **No** | Kopia uses the S3 backend. A new Kopia repo is created alongside Restic data. |
| **AWS S3 / other S3-compatible** | **No** | Same as above. |
| **LVP HostPath/NFS/PVC** (`replicated.com/hostpath`, `replicated.com/nfs`, `replicated.com/pvc`) | **Yes** | Kopia cannot use the LVP plugin. Reconfigure to the S3-compatible filesystem Minio path. |
| **kURL internal PVC** (`replicated.com/pvc`) | **Yes** | Reconfigure to the kURL S3-compatible internal store. |

- **In-product guidance:** KOTS shows an informational banner when an LVP-backed Restic BSL is detected on Velero 1.17+, explaining that new FSB backups require a Kopia-compatible BSL while existing Restic backups remain restorable.
- **Install command changes:** KOTS shows the correct command for the detected Velero version.

| Velero version | Install command |
| :---- | :---- |
| **< 1.10** | `--use-restic --use-volume-snapshots=false` |
| **1.10–1.16** | `--use-node-agent --uploader-type=restic --use-volume-snapshots=false` |
| **1.17+** | `--use-node-agent --use-volume-snapshots=false` (Kopia is the default) |

### New customers

The destination picker stays the same (AWS, GCP, Azure, S3-compatible, Internal, NFS, HostPath) but becomes **Velero-version-aware**. For Velero 1.17+, HostPath/NFS are routed to the S3-compatible filesystem Minio path instead of the LVP plugin. Customers who choose to exclude Minio due to its AGPL license will lose HostPath/NFS support and must use an object-store destination instead.

## What we will change

The core change is to make KOTS **Velero-version-aware**: install instructions, the destination picker, and the generated BackupStorageLocation config must all adapt to the installed Velero version. The LVP plugin is not used for new Kopia-era file-system backups; HostPath/NFS/PVC on Velero 1.17+ are served by the S3-compatible filesystem Minio path.

### 1\. KOTS repo (`replicatedhq/kots`)

| File | Change |
| :---- | :---- |
| `pkg/print/velero.go` | Version-aware install instructions and upgrade guidance. |
| `web/src/components/snapshots/ConfigureSnapshots.jsx` | Add `isVelero17OrNewer()` and return the correct flags. |
| `web/src/components/snapshots/SnapshotStorageDestination.tsx` | Make the destination picker version-aware; route 1.17+ HostPath/NFS/Internal to S3-compatible backends. |
| `pkg/snapshot/store.go` | For Velero 1.17+, create S3-compatible Minio BSLs for HostPath/NFS/PVC; omit LVP providers; preserve `resticRepoPrefix` on existing BSLs. |
| `cmd/kots/cli/velero.go` | Remove the unused `resticRepoBase` constant; version-aware CLI commands. |
| `e2e/velero/cli.go` | Remove hard-coded `--uploader-type=restic`. |
| `e2e/playwright/regression/shared/cli.ts` | Version-aware install helper. |
| `e2e/scripts/deps.sh` | Parameterized Velero version for CI matrix. |
| `pkg/kotsadmsnapshot/logparser_test.go` | Add Kopia log fixtures. |

### 2\. kURL repo (`replicatedhq/kURL`)

A minimal description of the required changes:

- **Add Velero 1.17.x and 1.18.x as add-on versions:** create new directories under `addons/velero/`, update `web/src/installers/versions.js` so the installer config screen offers the new versions, and run the existing `cron-velero-update` generator or manually mirror the 1.16.2 add-on structure.  
- **Update the Velero add-on install template:** in `addons/velero/template/base/install.tmpl.sh`, change the hard-coded `--use-node-agent --uploader-type=restic` to be version-aware (use the same version brackets as KOTS). For Velero 1.17+, use `--use-node-agent` only.  
- **Update the timeout flag mapping:** the template currently maps `resticTimeout` to `--restic-timeout`. For Velero 1.17+, map the same config value to `--fs-backup-timeout` (the Velero 1.17+ equivalent). The user-facing `resticTimeout` field can keep its name for backward compatibility.  
- **Update the config screen text:** the kURL installer UI that renders the Velero version list and the `disableRestic` / `resticTimeout` options should clarify that these control the file-system backup uploader (Restic or Kopia) for the selected Velero version. No new fields are required.  
- **Handle the read-only root filesystem case:** if the kURL installer uses a strict security context, the template should add `emptyDir` volumes for Kopia cache/config directories (`/home/cnb/udmrepo` and `/home/cnb/.cache`) when Velero 1.17+ is selected, because Kopia needs writable pod-local directories.

### 3\. LVP plugin repo (`replicatedhq/local-volume-provider`)

No code changes. Update the README and examples to state that LVP is not compatible with Kopia file-system backups on Velero 1.17+ and recommend S3-compatible storage (e.g., Minio) for local FSB.

### 4\. Docs repo (`replicated-docs`)

Update Velero compatibility, install, storage-destination, troubleshooting, and upgrade docs to reflect Kopia support for 1.17+, Restic preservation for older versions, and the Minio requirement for HostPath/NFS on 1.17+.

## High-level release plan

The work can be delivered as four independent, backward-compatible releases. Each release should be tested against both Restic and Kopia, and against the previous versions of the other repos, to ensure no cross-repo breakage.

| \# | Repo | Goal | What changes | Depends on | Backward compatibility | Rollback |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| 1 | `kots` | KOTS emits correct install flags for any Velero version | Version-aware instructions in `pkg/print`, `ConfigureSnapshots.jsx`, and e2e helpers | None | Keeps Restic flags for Velero \< 1.17 | Roll back KOTS binary |
| 2 | `kots` \+ docs | KOTS generates Kopia-compatible BSL config and documents the LVP limitation | Conditional `resticRepoPrefix` in `pkg/snapshot/store.go`; use filesystem Minio for HostPath/NFS/PVC on Velero 1.17+; docs update stating LVP is not supported for Kopia FSB | None | Existing Restic BSLs are untouched | Roll back KOTS binary |
| 3 | `kURL` | kURL offers and installs Velero 1.17/1.18 | New Velero add-on versions, version-aware template flags, config screen, timeout flag mapping | None | Older Velero add-on versions remain available | Pin installer to a previous Velero add-on version |
| 4 | `local-volume-provider` (docs-only) | Document Kopia incompatibility | README/examples updates stating LVP is not supported for Kopia FSB; no code changes | None | Restic behavior unchanged | Revert docs |

**Notes on independence:**

- Release 1 (KOTS install flags) can ship first.  
- Release 2 (KOTS BSL config + docs) can ship independently of the LVP docs update.  
- Release 3 (kURL) can ship independently of the LVP docs update.  
- Release 4 (LVP docs) is documentation-only and can ship at any time.  
- Releases 1 and 2 can be combined into a single KOTS release if that is simpler for the team.

## Communication plan

1. **Release notes:** Announce Velero 1.17/1.18 (Kopia) support, Restic preservation, and the LVP/Minio trade-off.
2. **Docs:** Update compatibility, install, storage-destination, troubleshooting, and upgrade docs. Include a migration page explaining version-by-version upgrades and the Minio requirement for HostPath/NFS on 1.17+.
3. **In-product messages:** The Admin Console automatically shows the correct Velero install command. Show an informational banner for LVP-backed Restic BSLs on 1.17+.
4. **Support enablement:** Provide a Community article covering Kopia vs Restic, `resticRepoPrefix`, the read-only-root-filesystem workaround, and LVP incompatibility.

## How we will be confident the result works for both Velero 1.16 and 1.17/1.18

| Confidence measure | How we will do it |
| :---- | :---- |
| **Dual-version install commands** | Run e2e smoke tests with Velero `v1.16.x` and `v1.18.x` and verify `velero install` exits 0 in both cases. |
| **Restic regression** | Run backup/restore with AWS and LVP on Velero 1.16. Confirm Restic repositories and restores still work. |
| **Kopia backup/restore** | Run backup/restore on Velero 1.18 with AWS S3, S3-compatible, and filesystem Minio (HostPath/NFS). Confirm Kopia repositories and restores work. |
| **Backward compatibility** | Create a Restic backup on Velero 1.16, upgrade to 1.17, and restore it. |
| **kURL add-on versions** | Add 1.17.x and 1.18.x to the kURL testgrid and run at least one install smoke test per version. |
| **Airgap image list** | Verify all required images (Velero, AWS plugin, LVP plugin for Restic, restore-helper, Minio) are pre-pushed for both versions. |

## Risks and mitigations

| Risk | Mitigation |
| :---- | :---- |
| Removing `resticRepoPrefix` from existing BSLs breaks Restic restores. | Only omit it for new Kopia-era stores; preserve the existing key on Restic-era stores. |
| Kopia data mover pods fail with read-only root filesystem. | Add `emptyDir` volumes for Kopia cache/config when strict security contexts are used. Document the workaround. |
| Customers try to use LVP HostPath/NFS/PVC for Kopia FSB and backups fail. | Document clearly that LVP is not compatible with Kopia FSB; ensure KOTS does not create new LVP-backed BSLs for Velero 1.17+; route HostPath/NFS/PVC destinations to the S3-compatible filesystem Minio path. |
| HostPath/NFS on 1.17+ fails because the filesystem Minio image is not available. | Include the Minio image in the KOTS/airgap bundle; ensure `ConfigureStore` deploys or validates Minio is present before creating a Minio-backed BSL on Velero 1.17+. |
| Customers exclude Minio due to AGPL licensing concerns and lose HostPath/NFS backup support. | Document clearly that Minio is required for HostPath/NFS on Velero 1.17+; provide alternative storage options (S3, GCP, Azure, S3-compatible); consider an opt-in/opt-out flag for filesystem Minio. |
| kURL install template still emits `--uploader-type=restic` for 1.17+. | Update the template to branch on `VELERO_VERSION`. |
| Customers are confused by “Restic” vs “Kopia” terminology. | Docs and in-product text use generic “file-system backup” language, with version-specific notes. |
| e2e matrix doubles runtime. | Run the full dual-version matrix only on snapshot-related changes; keep the default PR pin on a recent version. |

## Technical details (for implementation)

### Why the LVP plugin cannot be used for Kopia

Velero 1.17+ does not use object-store plugins for Kopia repository operations. It uses its own built-in backends (`s3`, `azure`, `gcs`, and `filesystem`). The LVP plugin registers `replicated.com/hostpath`, `replicated.com/nfs`, and `replicated.com/pvc`, none of which are recognized by Kopia. Therefore, Kopia cannot create a repository against an LVP BSL.

### How KOTS decides whether a BSL is Restic-era

KOTS uses the existing BSL object as the source of truth:

- **When re-configuring:** preserve `resticRepoPrefix` if it exists; otherwise do not add it.
- **When creating a new BSL:** include `resticRepoPrefix` on Velero < 1.17; omit it on Velero >= 1.17.

`ConfigureStoreOptions` should carry the existing BSL config (if any) and the target Velero version. The backend can detect the version via `snapshot.DetectVelero` or receive it from the frontend.

### HostPath/NFS/PVC on Velero 1.17+

On Velero 1.17+, KOTS deploys `kotsadm-fs-minio` on the selected HostPath or NFS mount and configures the BSL as `provider: aws` with `s3Url`. Kopia maps this to its S3 backend.

**Filesystem Minio default today**

| Scenario | Default state |
| :---- | :---- |
| New `kots install` on a non-kURL cluster | Enabled (`--with-minio` defaults to `true`). |
| New kURL cluster with the Minio add-on | Enabled. |
| New kURL cluster without the Minio add-on | Disabled (LVP is used). |
| After `kubectl kots velero migrate-minio-to-local-volume-provider` | Disabled (LVP is used). |
| Upgrade with `--with-minio=false` | Disabled (LVP is used). |

For Velero 1.17+, KOTS must ensure the Minio image is available when a HostPath/NFS destination is selected, even on clusters where it is currently disabled. The Minio image must be included in airgap bundles, and `ConfigureStore` should use the Minio path on 1.17+ regardless of the current `IsMinioDisabled` value.

### Manual Velero upgrade steps

For customers on 1.16 upgrading to 1.17+:

1. Install the Velero 1.17+ CLI.
2. Update CRDs: `velero install --crds-only --dry-run -o yaml | kubectl apply -f -`
3. Replace the stale `--uploader-type=restic` flag with `--uploader-type=kopia` in the Velero deployment and node-agent DaemonSet.
4. Update Velero and plugin images to the target 1.17.x/1.18.x version.
5. Confirm with `velero version`.

KOTS surfaces these instructions in the Admin Console but does not automate them.

## References

- Detailed research findings: `proposals/velero_kopia_support_research.md`  
- KOTS snapshot packages: `pkg/snapshot`, `pkg/kotsadmsnapshot`  
- Local-volume-provider plugin (cloned for analysis): `/var/folders/4r/xl9dbxjd7m583dm3ppmk0lqh0000gn/T/opencode/local-volume-provider`  
- kURL repo (cloned for analysis): `/Users/xav/go/src/github.com/replicatedhq/kURL`  
- Docs repo (cloned for analysis): `/Users/xav/work/replicated-docs`  
- Velero 1.17 release notes: [https://github.com/vmware-tanzu/velero/releases/tag/v1.17.0](https://github.com/vmware-tanzu/velero/releases/tag/v1.17.0)  
- Velero 1.17 FSB docs: [https://velero.io/docs/v1.17/file-system-backup/](https://velero.io/docs/v1.17/file-system-backup/)  
- Velero Restic deprecation: [https://velero.io/docs/v1.17/file-system-backup/\#restic-deprecation](https://velero.io/docs/v1.17/file-system-backup/#restic-deprecation)
