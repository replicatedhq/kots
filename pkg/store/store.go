package store

import (
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/util"
)

func init() {
	// Register the cluster ID provider with k8sutil to avoid import cycles
	k8sutil.SetClusterIDProvider(GetClusterIDIfAvailable)
}

var (
	globalStore Store
)

func GetStore() Store {
	if util.IsUpgradeService() {
		panic("store cannot not be used in the upgrade service")
	}
	if globalStore == nil {
		panic("store not initialized")
	}
	return globalStore
}

func SetStore(s Store) {
	if s == nil {
		globalStore = nil
		return
	}
	globalStore = s
}

// GetClusterIDIfAvailable returns the cluster ID from the store if initialized,
// or empty string if not. This allows callers to use the cluster ID as a fallback
// without panicking when the store isn't available (e.g., in CLI contexts).
func GetClusterIDIfAvailable() string {
	if util.IsUpgradeService() {
		return ""
	}
	if globalStore == nil {
		return ""
	}
	return globalStore.GetClusterID()
}
