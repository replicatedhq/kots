module github.com/replicatedhq/kots

go 1.12

require (
	cloud.google.com/go v0.52.0 // indirect
	github.com/14rcole/gopopulate v0.0.0-20180821133914-b175b219e774 // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/semver/v3 v3.0.2
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Masterminds/sprig/v3 v3.0.1
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/appscode/jsonpatch v1.0.1 // indirect
	github.com/aws/aws-sdk-go v1.25.18
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/chzyer/logex v1.1.11-0.20160617073814-96a4d311aa9b // indirect
	github.com/containerd/continuity v0.0.0-20191127005431-f65d91d395eb // indirect
	github.com/containers/image v3.0.2+incompatible
	github.com/containers/storage v1.15.8 // indirect
	github.com/coreos/etcd v3.3.15+incompatible // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/etcd-io/bbolt v1.3.3 // indirect
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/gobuffalo/flect v0.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.3.3 // indirect
	github.com/google/gofuzz v1.1.0
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.2.1 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/manifoldco/promptui v0.3.2
	github.com/mattn/go-isatty v0.0.9
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mtrmac/gpgme v0.0.0-20170102180018-b2432428689c // indirect
	github.com/nicksnyder/go-i18n v0.0.0-00010101000000-000000000000 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/ostreedev/ostree-go v0.0.0-20190702140239-759a8c1ac913 // indirect
	github.com/otiai10/copy v1.0.2
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.8 // indirect
	github.com/replicatedhq/kurl/kurlkinds v0.0.0-20200226193951-fe4b4890a55f
	github.com/replicatedhq/troubleshoot v0.9.21
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/tj/go-spin v1.1.0
	github.com/vbauerster/mpb v3.4.0+incompatible // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.1.0 // indirect
	go.uber.org/atomic v1.5.1 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0 // indirect
	go.undefinedlabs.com/scopeagent v0.1.7
	golang.org/x/crypto v0.0.0-20200204104054-c9f3fb736b72
	golang.org/x/lint v0.0.0-20200130185559-910be7a94367 // indirect
	golang.org/x/sys v0.0.0-20200202164722-d101bd2416d5 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20200203222849-174f5c63c9f5 // indirect
	gopkg.in/alecthomas/kingpin.v3-unstable v3.0.0-20180810215634-df19058c872c // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.2
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.3
	k8s.io/cli-runtime v0.17.0
	k8s.io/client-go v0.17.2
	k8s.io/helm v2.14.3+incompatible
	k8s.io/kube-openapi v0.0.0-20200130172213-cdac1c71ff9f // indirect
	k8s.io/kubernetes v1.13.0
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/kustomize/api v0.3.2
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65

replace github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
