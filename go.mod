module github.com/replicatedhq/kotsadm

go 1.12

require (
	github.com/andrewchambers/go-jqpipe v0.0.0-20180509223707-2d54cef8cd94 // indirect
	github.com/aws/aws-sdk-go v1.25.18
	github.com/containers/image v3.0.2+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0
	github.com/golang/mock v1.3.1 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.0
	github.com/kubernetes-sigs/application v0.8.1 // indirect
	github.com/lib/pq v1.3.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/kots v1.12.3-0.20200221022115-9cc0a8b3591c
	github.com/replicatedhq/kotsadm/kurl_proxy v0.0.0-20200221174232-bf8603192877 // indirect
	github.com/replicatedhq/kotsadm/operator v0.0.0-20200102212257-90833373b196
	github.com/replicatedhq/troubleshoot v0.9.21
	github.com/rogpeppe/fastuuid v1.1.0 // indirect
	github.com/segmentio/ksuid v1.0.2
	github.com/sergi/go-diff v1.0.0
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.6.1
	github.com/stretchr/testify v1.4.0
	github.com/vmware-tanzu/velero v1.2.0
	github.com/xo/dburl v0.0.0-20190203050942-98997a05b24f // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200204104054-c9f3fb736b72
	gopkg.in/go-playground/assert.v1 v1.2.1
	k8s.io/api v0.17.2
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/cli-runtime v0.17.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/application v0.8.1
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65

replace github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1

replace k8s.io/client-go => k8s.io/client-go v0.17.2
