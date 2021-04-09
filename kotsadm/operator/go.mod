module github.com/replicatedhq/kots/kotsadm/operator

go 1.14

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/google/martian v2.1.0+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/mitchellh/hashstructure v1.1.0
	github.com/pact-foundation/pact-go v1.5.1
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/troubleshoot v0.10.12
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
