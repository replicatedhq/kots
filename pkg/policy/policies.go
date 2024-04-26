package policy

/*
A Policy includes both an Action and a Resource.
An Action must be one of "read" or "write".
The Resource must use dot separators.
Permissions defined by Roles will define rules for both Action and Resource and may use glob
pattern matching.
*/

var (
	ActionRead  = "read"
	ActionWrite = "write"
)

// Redactor

var (
	RedactorRead  = Must(NewPolicy(ActionRead, "redactor."))
	RedactorWrite = Must(NewPolicy(ActionWrite, "redactor."))
)

// Registry

var (
	RegistryRead = Must(NewPolicy(ActionRead, "registry."))
)

// Snapshots

var (
	BackupRead            = Must(NewPolicy(ActionRead, "backup."))
	BackupWrite           = Must(NewPolicy(ActionWrite, "backup."))
	RestoreRead           = Must(NewPolicy(ActionRead, "restore."))
	RestoreWrite          = Must(NewPolicy(ActionWrite, "restore."))
	SnapshotsettingsRead  = Must(NewPolicy(ActionRead, "snapshotsettings."))
	SnapshotsettingsWrite = Must(NewPolicy(ActionWrite, "snapshotsettings."))
)

// Cluster

var (
	ClusterRead  = Must(NewPolicy(ActionRead, "cluster."))
	ClusterWrite = Must(NewPolicy(ActionWrite, "cluster."))
)

// GitOps

var (
	GitOpsRead  = Must(NewPolicy(ActionRead, "gitops."))
	GitOpsWrite = Must(NewPolicy(ActionWrite, "gitops."))
)

// Prometheus

var (
	PrometheussettingsWrite = Must(NewPolicy(ActionWrite, "prometheussettings."))
)

// Password change

var (
	PasswordChange = Must(NewPolicy(ActionWrite, "passwordupdate."))
)

// Kotsadm Identity Service

var (
	IdentityServiceWrite = Must(NewPolicy(ActionWrite, "identityservice."))
	IdentityServiceRead  = Must(NewPolicy(ActionRead, "identityservice."))
)

// App Identity Service
var (
	AppIdentityServiceWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.identityservice."))
	AppIdentityServiceRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.identityservice."))
)

// App

var (
	AppList   = Must(NewPolicy(ActionRead, "app."))
	AppRead   = Must(NewPolicy(ActionRead, "app.{{.appSlug}}"))
	AppCreate = Must(NewPolicy(ActionWrite, "app."))
	AppUpdate = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}", appSlugFromAppIDGetter))
)

// App status

var (
	AppStatusRead = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.status."))
)

// App supportbundle

var (
	AppSupportbundleRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.supportbundle.", appSlugFromSupportbundleGetter))
	AppSupportbundleWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.supportbundle.", appSlugFromAppIDGetter))
)

// App snapshots

var (
	AppBackupRead            = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.backup."))
	AppBackupWrite           = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.backup."))
	AppRestoreRead           = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.restore."))
	AppRestoreWrite          = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.restore."))
	AppSnapshotsettingsRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.snapshotsettings."))
	AppSnapshotsettingsWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.snapshotsettings."))
)

// App registry

var (
	AppRegistryRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.registry."))
	AppRegistryWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.registry."))
)

// App license

var (
	AppLicenseRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.license."))
	AppLicenseWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.license."))
)

// App gitops

var (
	AppGitopsRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.gitops.", appSlugFromAppIDGetter))
	AppGitopsWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.gitops.", appSlugFromAppIDGetter))
)

// App downstream

var (
	AppDownstreamRead         = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.downstream."))
	AppDownstreamWrite        = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.downstream."))
	AppDownstreamLogsRead     = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.downstream.logs."))
	AppDownstreamFiletreeRead = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.downstream.filetree."))
)

// App downstream preflight

var (
	AppDownstreamPreflightRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.downstream.preflight."))
	AppDownstreamPreflightWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.downstream.preflight."))
)

// App downstream config

var (
	AppDownstreamConfigRead  = Must(NewPolicy(ActionRead, "app.{{.appSlug}}.downstream.config."))
	AppDownstreamConfigWrite = Must(NewPolicy(ActionWrite, "app.{{.appSlug}}.downstream.config."))
)
