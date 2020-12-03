module github.com/replicatedhq/kots/kotsadm

go 1.14

require (
	cloud.google.com/go v0.46.2
	github.com/Azure/azure-sdk-for-go v42.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/adal v0.8.2
	github.com/aws/aws-sdk-go v1.28.2
	github.com/bitnami-labs/sealed-secrets v0.12.5
	github.com/containerd/containerd v1.3.2
	github.com/containers/image/v5 v5.5.2
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/deislabs/oras v0.8.1
	github.com/dexidp/dex v0.0.0-20201105145354-71bbbee07527
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-units v0.4.0
	github.com/emicklei/go-restful v2.10.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-logfmt/logfmt v0.4.0
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.0
	github.com/gosimple/slug v1.9.0
	github.com/gtank/cryptopasta v0.0.0-20170601214702-1f550f6f2f69 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.0
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/lib/pq v1.3.0
	github.com/marccampbell/yaml-toolbox v0.0.0-20200805160637-950ceb36c770
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/open-policy-agent/opa v0.24.0 // indirect
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/kots v0.0.0-00010101000000-000000000000
	github.com/replicatedhq/troubleshoot v0.9.52
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/robfig/cron v1.1.0
	github.com/robfig/cron/v3 v3.0.0
	github.com/segmentio/ksuid v1.0.3
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.6.1
	github.com/vmware-tanzu/velero v1.5.1
	go.uber.org/multierr v1.3.0
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200319210407-521f4a0cd458 // indirect
	google.golang.org/api v0.15.0
	gopkg.in/ini.v1 v1.51.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/cluster-bootstrap v0.18.4
	k8s.io/kubernetes v1.18.4
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19
	sigs.k8s.io/application v0.8.2
	sigs.k8s.io/controller-runtime v0.6.1
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65
	github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
	github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
	github.com/replicatedhq/kots => ../
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
	k8s.io/api => k8s.io/api v0.18.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.4
	k8s.io/apiserver => k8s.io/apiserver v0.18.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.4
	k8s.io/client-go => k8s.io/client-go v0.18.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.4
	k8s.io/code-generator => k8s.io/code-generator v0.18.4
	k8s.io/component-base => k8s.io/component-base v0.18.4
	k8s.io/cri-api => k8s.io/cri-api v0.18.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.4
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190822140433-26a664648505
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1
	k8s.io/klog => k8s.io/klog v1.0.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.4
	k8s.io/kubectl => k8s.io/kubectl v0.18.4
	k8s.io/kubelet => k8s.io/kubelet v0.18.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.4
	k8s.io/metrics => k8s.io/metrics v0.18.4
	k8s.io/node-api => k8s.io/node-api v0.18.4
	k8s.io/repo-infra => k8s.io/repo-infra v0.0.1-alpha.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.18.4
	k8s.io/sample-controller => k8s.io/sample-controller v0.18.4
	k8s.io/utils => k8s.io/utils v0.0.0-20200603063816-c1c6865ac451
)
