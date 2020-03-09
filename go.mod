module github.com/replicatedhq/kots

go 1.12

require (
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/semver/v3 v3.0.2
	github.com/Masterminds/sprig/v3 v3.0.1
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/aws/aws-sdk-go v1.25.18
	github.com/containerd/continuity v0.0.0-20200228182428-0f16d7a0959c // indirect
	github.com/containers/image v3.0.2+incompatible
	github.com/containers/storage v1.16.2 // indirect
	github.com/coreos/etcd v3.3.15+incompatible // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-units v0.4.0
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/fatih/color v1.7.0
	github.com/frankban/quicktest v1.7.2 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/gofuzz v1.1.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/manifoldco/promptui v0.3.2
	github.com/mattn/go-isatty v0.0.9
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mtrmac/gpgme v0.1.2 // indirect
	github.com/otiai10/copy v1.0.2
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/troubleshoot v0.9.26
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	github.com/tj/go-spin v1.1.0
	github.com/ugorji/go v1.1.7 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.undefinedlabs.com/scopeagent v0.1.7
	golang.org/x/crypto v0.0.0-20200204104054-c9f3fb736b72
	gopkg.in/alecthomas/kingpin.v3-unstable v3.0.0-20191105091915-95d230a53780 // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.2
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/cli-runtime v0.17.0
	k8s.io/client-go v0.17.2
	k8s.io/helm v2.14.3+incompatible
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/kustomize/api v0.3.2
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65

replace github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
