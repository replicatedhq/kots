module github.com/replicatedhq/kots/kotsadm/operator

go 1.16

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/frankban/quicktest v1.13.0 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/martian v2.1.0+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/hashstructure v1.1.0
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/pact-foundation/pact-go v1.5.3
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/troubleshoot v0.13.7
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/oauth2 v0.0.0-20210622215436-a8dc77f794b6 // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/klog/v2 v2.9.0 // indirect
	sigs.k8s.io/controller-runtime v0.9.5
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
)

replace (
	github.com/longhorn/longhorn-manager => github.com/replicatedhq/longhorn-manager v1.1.2-0.20210622201804-05b01947b99d
	github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
	k8s.io/api => k8s.io/api v0.20.5
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.5
	k8s.io/apiserver => k8s.io/apiserver v0.20.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.5
	k8s.io/client-go => k8s.io/client-go v0.20.5
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.5
	k8s.io/code-generator => k8s.io/code-generator v0.20.5
	k8s.io/component-base => k8s.io/component-base v0.20.5
	k8s.io/component-helpers => k8s.io/component-helpers v0.20.5
	k8s.io/controller-manager => k8s.io/controller-manager v0.20.5
	k8s.io/cri-api => k8s.io/cri-api v0.20.5
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.5
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.5
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.5
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.5
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.5
	k8s.io/kubectl => k8s.io/kubectl v0.20.5
	k8s.io/kubelet => k8s.io/kubelet v0.20.5
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.5
	k8s.io/metrics => k8s.io/metrics v0.20.5
	k8s.io/mount-utils => k8s.io/mount-utils v0.20.5
	k8s.io/node-api => k8s.io/node-api v0.20.5
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.5
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.5
	k8s.io/sample-controller => k8s.io/sample-controller v0.20.5
)
