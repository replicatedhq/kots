module github.com/replicatedhq/kots

go 1.16

require (
	cloud.google.com/go/storage v1.10.0
	github.com/Azure/azure-sdk-for-go v55.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/ahmetalpbalkan/go-cursor v0.0.0-20131010032410-8136607ea412
	github.com/aws/aws-sdk-go v1.38.49
	github.com/bitnami-labs/sealed-secrets v0.14.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/containerd v1.5.5
	github.com/containers/image/v5 v5.15.2
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dexidp/dex v0.0.0-20201105145354-71bbbee07527
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/go-units v0.4.0
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/color v1.12.0
	github.com/frankban/quicktest v1.13.0 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-logfmt/logfmt v0.5.0
	github.com/go-test/deep v1.0.7
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/mock v1.5.0
	github.com/google/go-github/v39 v39.0.0
	github.com/google/gofuzz v1.2.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/gosimple/slug v1.9.0
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/k3s-io/kine v0.7.3
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lib/pq v1.10.2
	github.com/manifoldco/promptui v0.8.0
	github.com/marccampbell/yaml-toolbox v0.0.0-20200805160637-950ceb36c770
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/hashstructure v1.1.0
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/open-policy-agent/opa v0.24.0
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/openshift/api v0.0.0-20210513192832-efee9960e6fd // indirect
	github.com/openshift/client-go v0.0.0-20210503124028-ac0910aac9fa
	github.com/otiai10/copy v1.0.2
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pierrec/lz4 v2.2.6+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/rancher/wrangler v0.8.3
	github.com/replicatedhq/kurl v0.0.0-20210414162418-8d6211901244
	github.com/replicatedhq/troubleshoot v0.14.0
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/robfig/cron v1.2.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rubenv/sql-migrate v0.0.0-20210614095031-55d5740dbbcc // indirect
	github.com/russellhaering/goxmldsig v1.1.0 // indirect
	github.com/schemahero/schemahero v0.12.2
	github.com/segmentio/ksuid v1.0.3
	github.com/sergi/go-diff v1.1.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stevvooe/resumable v0.0.0-20180830230917-22b14a53ba50 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tj/go-spin v1.1.0
	github.com/vmware-tanzu/velero v1.5.1
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect; indirect=
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	google.golang.org/api v0.44.0
	google.golang.org/grpc v1.38.0
	gopkg.in/go-playground/assert.v1 v1.2.1
	gopkg.in/ini.v1 v1.62.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.6.1-0.20210819153322-82a2abf51252
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/cli-runtime v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/cluster-bootstrap v0.22.1
	k8s.io/helm v2.14.3+incompatible
	k8s.io/klog/v2 v2.9.0
	k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8 // indirect
	k8s.io/kubectl v0.22.1 // indirect
	k8s.io/kubelet v0.0.0
	k8s.io/kubernetes v1.22.1
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	oras.land/oras-go v0.4.0
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/application v0.8.3
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/kustomize/api v0.8.11
	sigs.k8s.io/kustomize/kyaml v0.11.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65
	github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
	github.com/go-openapi/jsonpointer => github.com/go-openapi/jsonpointer v0.19.5
	github.com/go-openapi/jsonreference => github.com/go-openapi/jsonreference v0.19.6
	github.com/go-openapi/loads => github.com/go-openapi/loads v0.20.1
	github.com/go-openapi/runtime => github.com/go-openapi/runtime v0.19.30
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.20.1
	github.com/go-openapi/strfmt => github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag => github.com/go-openapi/swag v0.19.15
	github.com/go-openapi/validate => github.com/go-openapi/validate v0.20.1
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.5
	github.com/jmoiron/sqlx v1.2.0 => github.com/longquanzheng/sqlx v0.0.0-20191125235044-053e6130695c
	github.com/longhorn/longhorn-manager => github.com/replicatedhq/longhorn-manager v1.1.2-0.20210622201804-05b01947b99d
	google.golang.org/grpc => google.golang.org/grpc v1.38.0
	gopkg.in/square/go-jose.v2 => gopkg.in/square/go-jose.v2 v2.2.2
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.4.0
	k8s.io/api => github.com/k3s-io/kubernetes/staging/src/k8s.io/api v1.22.1-k3s1
	k8s.io/apiextensions-apiserver => github.com/k3s-io/kubernetes/staging/src/k8s.io/apiextensions-apiserver v1.22.1-k3s1
	k8s.io/apimachinery => github.com/k3s-io/kubernetes/staging/src/k8s.io/apimachinery v1.22.1-k3s1
	k8s.io/apiserver => github.com/k3s-io/kubernetes/staging/src/k8s.io/apiserver v1.22.1-k3s1
	k8s.io/cli-runtime => github.com/k3s-io/kubernetes/staging/src/k8s.io/cli-runtime v1.22.1-k3s1
	k8s.io/client-go => github.com/k3s-io/kubernetes/staging/src/k8s.io/client-go v1.22.1-k3s1
	k8s.io/cloud-provider => github.com/k3s-io/kubernetes/staging/src/k8s.io/cloud-provider v1.22.1-k3s1
	k8s.io/cluster-bootstrap => github.com/k3s-io/kubernetes/staging/src/k8s.io/cluster-bootstrap v1.22.1-k3s1
	k8s.io/code-generator => github.com/k3s-io/kubernetes/staging/src/k8s.io/code-generator v1.22.1-k3s1
	k8s.io/component-base => github.com/k3s-io/kubernetes/staging/src/k8s.io/component-base v1.22.1-k3s1
	k8s.io/component-helpers => github.com/k3s-io/kubernetes/staging/src/k8s.io/component-helpers v1.22.1-k3s1
	k8s.io/controller-manager => github.com/k3s-io/kubernetes/staging/src/k8s.io/controller-manager v1.22.1-k3s1
	k8s.io/cri-api => github.com/k3s-io/kubernetes/staging/src/k8s.io/cri-api v1.22.1-k3s1
	k8s.io/csi-translation-lib => github.com/k3s-io/kubernetes/staging/src/k8s.io/csi-translation-lib v1.22.1-k3s1
	k8s.io/kube-aggregator => github.com/k3s-io/kubernetes/staging/src/k8s.io/kube-aggregator v1.22.1-k3s1
	k8s.io/kube-controller-manager => github.com/k3s-io/kubernetes/staging/src/k8s.io/kube-controller-manager v1.22.1-k3s1
	k8s.io/kube-proxy => github.com/k3s-io/kubernetes/staging/src/k8s.io/kube-proxy v1.22.1-k3s1
	k8s.io/kube-scheduler => github.com/k3s-io/kubernetes/staging/src/k8s.io/kube-scheduler v1.22.1-k3s1
	k8s.io/kubectl => github.com/k3s-io/kubernetes/staging/src/k8s.io/kubectl v1.22.1-k3s1
	k8s.io/kubelet => github.com/k3s-io/kubernetes/staging/src/k8s.io/kubelet v1.22.1-k3s1
	k8s.io/kubernetes => github.com/k3s-io/kubernetes v1.22.1-k3s1
	k8s.io/kubernetes/pkg/serviceaccount => github.com/k3s-io/kubernetes/staging/src/k8s.io/kubernetes/pkg/serviceaccount v1.22.1-k3s1
	k8s.io/legacy-cloud-providers => github.com/k3s-io/kubernetes/staging/src/k8s.io/legacy-cloud-providers v1.22.1-k3s1
	k8s.io/metrics => github.com/k3s-io/kubernetes/staging/src/k8s.io/metrics v1.22.1-k3s1
	k8s.io/mount-utils => github.com/k3s-io/kubernetes/staging/src/k8s.io/mount-utils v1.22.1-k3s1
	k8s.io/node-api => github.com/k3s-io/kubernetes/staging/src/k8s.io/node-api v1.22.1-k3s1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.1
	k8s.io/sample-apiserver => github.com/k3s-io/kubernetes/staging/src/k8s.io/sample-apiserver v1.22.1-k3s1
	k8s.io/sample-cli-plugin => github.com/k3s-io/kubernetes/staging/src/k8s.io/sample-cli-plugin v1.22.1-k3s1
	k8s.io/sample-controller => github.com/k3s-io/kubernetes/staging/src/k8s.io/sample-controller v1.22.1-k3s1
)
