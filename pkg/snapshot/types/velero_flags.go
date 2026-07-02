package types

import (
	"github.com/Masterminds/semver/v3"
)

// ErrLVPUnsupportedOnVelero117 is the canonical message returned when a
// filesystem (NFS / Host Path / internal-store-without-minio) snapshot store is
// being configured against a Velero version that no longer supports the
// local-volume-provider plugin.
const ErrLVPUnsupportedOnVelero117 = "filesystem (NFS/Host Path) snapshots via the local-volume-provider plugin are not supported on Velero 1.17+; use an object-store backend or a Velero version < 1.17"

var (
	// velero110 is the first Velero release with the node-agent + uploader-type
	// model (replacing the standalone restic daemonset).
	velero110 = semver.MustParse("1.10.0")
	// velero117 is the first Velero release that removed the restic uploader;
	// kopia is the only valid uploader type from this version on, and the
	// local-volume-provider plugin is no longer supported.
	velero117 = semver.MustParse("1.17.0")
)

// VeleroFSBackupFlags returns the file-system-backup install flags for the given
// Velero version:
//
//	< 1.10            -> --use-restic (no node-agent/uploader-type)
//	>= 1.10, < 1.17   -> --use-node-agent --uploader-type=restic
//	>= 1.17           -> --use-node-agent (kopia is the implicit default uploader)
//
// An empty or unparseable version is treated as a current (>= 1.17) release,
// since install instructions are shown before Velero exists in the cluster and
// users are told to install the latest Velero.
func VeleroFSBackupFlags(veleroVersion string) []string {
	v, err := parseVeleroVersion(veleroVersion)
	if err != nil { // empty or unparseable -> newest behavior
		return []string{"--use-node-agent"}
	}
	switch {
	case v.LessThan(velero110):
		return []string{"--use-restic"}
	case v.LessThan(velero117):
		return []string{"--use-node-agent", "--uploader-type=restic"}
	default:
		return []string{"--use-node-agent"}
	}
}

// VeleroSupportsLVP reports whether the local-volume-provider plugin is supported
// for the given Velero version. LVP is unsupported on Velero >= 1.17. An empty or
// unparseable version is treated as supported (unknown): the version is unknown
// until Velero is installed, and the configure path re-checks once it is detected.
func VeleroSupportsLVP(veleroVersion string) bool {
	v, err := parseVeleroVersion(veleroVersion)
	if err != nil { // unknown: don't block
		return true
	}
	return v.LessThan(velero117)
}

// parseVeleroVersion parses a Velero version string (tolerating a leading "v")
// and strips any pre-release / build metadata, so that the major.minor.patch
// release boundary is what decides behavior. Without this, semver precedence
// would rank e.g. "1.17.0-rc.1" below "1.17.0", classifying a 1.17 release
// candidate (which already dropped the restic uploader) as a pre-1.17 release.
func parseVeleroVersion(veleroVersion string) (*semver.Version, error) {
	v, err := semver.NewVersion(veleroVersion)
	if err != nil {
		return nil, err
	}
	core := semver.New(v.Major(), v.Minor(), v.Patch(), "", "")
	return core, nil
}
