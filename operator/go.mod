module github.com/replicatedhq/kotsadm/operator

go 1.12

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/etcd v3.3.15+incompatible // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/hashstructure v1.0.0
	github.com/pact-foundation/pact-go v1.0.0-beta.5
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/troubleshoot v0.9.26
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/ugorji/go v1.1.7 // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	go.undefinedlabs.com/scopeagent v0.1.12
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/helm v2.14.3+incompatible
)

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
