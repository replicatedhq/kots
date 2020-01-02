module github.com/replicatedhq/kotsadm

go 1.12

require (
	github.com/andrewchambers/go-jqpipe v0.0.0-20180509223707-2d54cef8cd94 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/golang/mock v1.3.1 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.0
	github.com/lib/pq v1.3.0
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/pierrec/lz4 v2.2.6+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/replicatedhq/kots v1.9.0
	github.com/replicatedhq/kotsadm/operator v0.0.0-20200102212257-90833373b196
	github.com/rogpeppe/fastuuid v1.1.0 // indirect
	github.com/segmentio/ksuid v1.0.2
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.6.1
	github.com/xo/dburl v0.0.0-20190203050942-98997a05b24f // indirect
	go.opencensus.io v0.20.2 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20190911031432-227b76d455e7
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.2.0-beta.2
	sigs.k8s.io/controller-tools v0.2.0-beta.2 // indirect
)

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v2.0.3+incompatible
