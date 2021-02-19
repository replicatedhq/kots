module github.com/replicatedhq/kots

go 1.14

require (
	cloud.google.com/go v0.46.2
	github.com/Azure/azure-sdk-for-go v42.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/adal v0.8.2
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Masterminds/sprig/v3 v3.2.0
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/aws/aws-sdk-go v1.28.2
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bitnami-labs/sealed-secrets v0.14.1
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v0.4.1 // indirect
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/containerd v1.3.2
	github.com/containerd/continuity v0.0.0-20200228182428-0f16d7a0959c // indirect
	github.com/containers/image/v5 v5.5.2
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/deislabs/oras v0.8.1
	github.com/dexidp/dex v0.0.0-20201105145354-71bbbee07527
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-units v0.4.0
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.7.0
	github.com/frankban/quicktest v1.7.2 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-logfmt/logfmt v0.5.0
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/golang/mock v1.4.4
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/gofuzz v1.1.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.0
	github.com/gosimple/slug v1.9.0
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lib/pq v1.3.0
	github.com/manifoldco/promptui v0.3.2
	github.com/marccampbell/yaml-toolbox v0.0.0-20200805160637-950ceb36c770
	github.com/mattn/go-isatty v0.0.12
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/open-policy-agent/opa v0.24.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/otiai10/copy v1.0.2
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pierrec/lz4 v2.2.6+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/kurl/kurlkinds v0.0.0-20200617224429-55f493a72baf
	github.com/replicatedhq/troubleshoot v0.9.55
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/robfig/cron v1.1.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/russellhaering/goxmldsig v1.1.0 // indirect
	github.com/segmentio/ksuid v1.0.3
	github.com/sergi/go-diff v1.1.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stevvooe/resumable v0.0.0-20180830230917-22b14a53ba50 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/tj/go-spin v1.1.0
	github.com/vmware-tanzu/velero v1.5.1
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	go.uber.org/multierr v1.3.0
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.15.0
	gopkg.in/go-playground/assert.v1 v1.2.1
	gopkg.in/ini.v1 v1.51.0
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.18.4
	k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/cli-runtime v0.18.4
	k8s.io/client-go v0.18.4
	k8s.io/cluster-bootstrap v0.17.8
	k8s.io/helm v2.14.3+incompatible
	k8s.io/kubernetes v1.18.0
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/application v0.8.2
	sigs.k8s.io/controller-runtime v0.6.1
	sigs.k8s.io/kustomize/api v0.3.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65
	github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
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
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.4
	k8s.io/kubectl => k8s.io/kubectl v0.18.4
	k8s.io/kubelet => k8s.io/kubelet v0.18.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.4
	k8s.io/metrics => k8s.io/metrics v0.18.4
	k8s.io/node-api => k8s.io/node-api v0.18.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.18.4
	k8s.io/sample-controller => k8s.io/sample-controller v0.18.4
)
