# Instance ID Regeneration on Snapshot Restore

Design note for ko-s5p (sc-133130). KOTS detects when an installation has been
restored from a snapshot into a different environment and regenerates the
reported `instance_id`, recording the restore lineage so the vendor portal can
link the new instance to the one it was restored from.

## Problem

The values KOTS reports as instance identity are durable across snapshot
restores:

- `instance_id` (`X-Replicated-InstanceID`) is the app ID, a primary key in the
  kotsadm database. The database is included in backups, so every environment
  restored from the same backup reports the same `instance_id`.
- `cluster_id` (`X-Replicated-ClusterID`) comes from the `kotsadm-id` ConfigMap.
  The ConfigMap is excluded from backups, but on startup the ID is re-derived
  from the restored database when the ConfigMap is missing.

Result: multiple environments restored from one snapshot all report as a single
instance (sc-130386: 3,000+ installations under one `instance_id`).

## Design

### Detection: environment fingerprint

A fingerprint of the environment is stored in the `kotsadm_params` table under
the key `ENVIRONMENT_FINGERPRINT`. Because `kotsadm_params` lives in the kotsadm
database, the fingerprint travels inside every backup. After a restore, the
stored fingerprint describes the environment the backup was taken in, and it is
compared against the live environment at kotsadm startup (`reporting.Init`).

The fingerprint is JSON with two fields:

| Field | Source | Why |
|---|---|---|
| `kubeSystemUID` | UID of the `kube-system` namespace | Canonical cluster identity. Stable for the lifetime of a cluster; never recreated in place. Changes exactly when the database lands in a different cluster. |
| `podNamespaceUID` | UID of the namespace kotsadm runs in | Fallback for namespace-scoped installs whose RBAC cannot read `kube-system`. |

Comparison rules, applied at startup:

1. No stored fingerprint → **adopt**: store the current fingerprint and keep
   the existing instance ID. This is the upgrade path for existing
   installations (gradual rollout: nothing regenerates on upgrade).
2. Both stored and current have `kubeSystemUID` → compare it. Equal → **keep**
   (same cluster). Different → **regenerate**.
3. Otherwise, both have `podNamespaceUID` → compare it. Equal → **keep**.
   Different → **regenerate**.
4. No comparable field → **keep** (fail safe: never regenerate when uncertain).

On **keep**, the stored fingerprint is refreshed with the current values so
secondary fields don't go stale. On **regenerate**, the new fingerprint is
stored only after all apps have been regenerated, so a crash mid-way re-runs
detection on next boot (a duplicate lineage entry is preferable to a missed
regeneration).

### Regeneration: decoupling the reported instance ID from the app ID

The app ID is a primary key with foreign keys throughout the database, so it is
not mutated. Instead, the reported instance ID is decoupled from it:

- Two new nullable columns on the `app` table: `instance_id` and
  `instance_id_lineage` (JSON array, oldest first).
- When `instance_id` is NULL, the reported instance ID is the app ID — existing
  and freshly-installed apps behave exactly as today.
- When a restore is detected, each installed app gets a new KSUID `instance_id`
  and the previous value is appended to `instance_id_lineage`.

### Reporting the lineage

`ReportingInfo` gains `InstanceID` resolution and a new
`RestoredFromInstanceID` field (the immediate parent, i.e. the last lineage
entry), sent as:

- `restored_from_instance_id` in instance report events (online and airgap)
- `X-Replicated-RestoredFromInstanceID` header on update-check requests

Sequential restores produce a chain (A → B → C); each generation reports its
immediate parent, so the full chain is reconstructable server-side.

## How each scenario resolves

| Scenario | Fingerprint behavior | Outcome |
|---|---|---|
| Normal restart / upgrade | unchanged | ID kept |
| IP address change (nodes, pods, LB) | fingerprint contains no addresses | ID kept |
| Full snapshot restore into the same, still-existing cluster (in-place DR) | `kubeSystemUID` unchanged | ID kept |
| Full snapshot restore into a different cluster | `kubeSystemUID` differs | new ID + lineage |
| Repeated restores into fresh clusters (CI) | differs each time | chain A → B → C |
| Original + restored running in parallel | restored env differs | distinct IDs |
| Existing install upgrading to this version | no stored fingerprint | adopt, ID kept |
| EC/kURL: restore onto a rebuilt/new cluster | `kubeSystemUID` differs | new ID + lineage |

## Known limitations

- **Bit-identical VM clones are not detectable client-side.** A VM snapshot
  restored in place, or cloned byte-for-byte, contains the same `kube-system`
  UID as the original. No in-cluster signal distinguishes such clones (IP/MAC
  changes are explicitly not safe signals — normal IP changes must not trigger
  regeneration). Deduplicating identical clones requires server-side
  correlation (e.g., detecting interleaved reports against the monotonic
  event timeline) and is out of scope for this repo.
- **DR onto a rebuilt cluster regenerates the ID.** "Same environment" is
  defined as "same Kubernetes cluster" (`kube-system` UID). If a cluster is
  destroyed and rebuilt for DR, the restored install is a new instance whose
  lineage points at the old one; the portal can present continuity via lineage.
- **Preflight reporting is unchanged.** The online preflight endpoint encodes
  the app ID in its URL path (server API contract), so preflight reports
  continue to use the app ID.
