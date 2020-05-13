module github.com/replicatedhq/kots/kotsadm/operator

go 1.14

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/mitchellh/hashstructure v1.0.0
	github.com/pact-foundation/pact-go v1.0.0-beta.5
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/troubleshoot v0.9.31
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	go.undefinedlabs.com/scopeagent v0.1.12
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.17.4
	sigs.k8s.io/controller-runtime v0.4.0
)

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
