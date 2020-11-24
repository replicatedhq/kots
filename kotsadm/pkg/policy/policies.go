package policy

// Support bundles

var SupportBundlesRead = Must(NewPolicy("/supportbundles/{{.bundleId}}/read"))

// Apps

var AppSupportBundlesList = Must(NewPolicy("/app/{{.appSlug}}/supportbundles/list"))
