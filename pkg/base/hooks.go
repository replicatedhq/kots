package base

// HookAnnotation is the label name for a hook
const HookAnnotation = "kots.io/hook"

// HookWeightAnnotation is the label name for a hook weight
const HookWeightAnnotation = "kots.io/hook-weight"

// HookDeleteAnnotation is the label name for the delete policy for a hook
const HookDeleteAnnotation = "kots.io/hook-delete-policy"

// HookEvent specifies the hook event
type HookEvent string

// Hook event types
const (
	HookPreInstall   HookEvent = "pre-install"
	HookPostInstall  HookEvent = "post-install"
	HookPreDelete    HookEvent = "pre-delete"
	HookPostDelete   HookEvent = "post-delete"
	HookPreUpgrade   HookEvent = "pre-upgrade"
	HookPostUpgrade  HookEvent = "post-upgrade"
	HookPreRollback  HookEvent = "pre-rollback"
	HookPostRollback HookEvent = "post-rollback"
	HookTest         HookEvent = "test"
)

func (x HookEvent) String() string { return string(x) }

var hookEvents = map[string]HookEvent{
	HookPreInstall.String():   HookPreInstall,
	HookPostInstall.String():  HookPostInstall,
	HookPreDelete.String():    HookPreDelete,
	HookPostDelete.String():   HookPostDelete,
	HookPreUpgrade.String():   HookPreUpgrade,
	HookPostUpgrade.String():  HookPostUpgrade,
	HookPreRollback.String():  HookPreRollback,
	HookPostRollback.String(): HookPostRollback,
	HookTest.String():         HookTest,
	// Support test-success for backward compatibility with Helm 2 tests
	"test-success": HookTest,
}
