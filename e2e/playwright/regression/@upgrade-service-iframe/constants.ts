// Placeholder constants for the upgrade-service iframe regression test.
// The test itself reads the upgrade version and optional hostname from environment
// variables so it can be reused against different embedded-cluster apps.

export const NAMESPACE = "default";
export const IS_AIRGAPPED = false;
export const IS_MINIMAL_RBAC = false;
export const IS_EC = true;
