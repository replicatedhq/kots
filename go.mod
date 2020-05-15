module github.com/replicatedhq/kots

go 1.14

require (
	github.com/14rcole/gopopulate v0.0.0-20180821133914-b175b219e774 // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/aws/aws-sdk-go v1.25.18
	github.com/containerd/continuity v0.0.0-20200228182428-0f16d7a0959c // indirect
	github.com/containers/image v3.0.2+incompatible
	github.com/containers/storage v1.16.2 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/etcd-io/bbolt v1.3.3 // indirect
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/gofuzz v1.1.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/manifoldco/promptui v0.3.2
	github.com/mattn/go-isatty v0.0.9
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mtrmac/gpgme v0.1.2 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/ostreedev/ostree-go v0.0.0-20190702140239-759a8c1ac913 // indirect
	github.com/otiai10/copy v1.0.2
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/kurl/kurlkinds v0.0.0-20200306230415-b6d377a48a56
	github.com/replicatedhq/troubleshoot v0.9.33
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	github.com/tj/go-spin v1.1.0
	github.com/vbauerster/mpb v3.4.0+incompatible // indirect
	go.undefinedlabs.com/scopeagent v0.1.12
	golang.org/x/crypto v0.0.0-20200414173820-0848c9571904
	gopkg.in/yaml.v2 v2.2.8
	helm.sh/helm/v3 v3.1.2
	k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.4
	k8s.io/cli-runtime v0.17.4
	k8s.io/client-go v0.17.4
	k8s.io/code-generator v0.18.3-beta.0 // indirect
	k8s.io/helm v2.14.3+incompatible
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/controller-tools v0.2.8 // indirect
	sigs.k8s.io/kustomize/api v0.3.2
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65

replace github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
