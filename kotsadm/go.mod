module github.com/replicatedhq/kots/kotsadm

go 1.14

require (
	cloud.google.com/go v0.38.0
	github.com/Azure/azure-sdk-for-go v35.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest/adal v0.5.0
	github.com/aws/aws-sdk-go v1.25.18
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v0.4.1 // indirect
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/containerd/containerd v1.3.2
	github.com/deislabs/oras v0.8.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/go-units v0.4.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/gorilla/handlers v1.4.2 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.0
	github.com/gosimple/slug v1.9.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/lib/pq v1.3.0
	github.com/marccampbell/yaml-toolbox v0.0.0-20200328202846-85b6f7184a20
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/kots v1.15.3-0.20200512202738-aaa332813ead
	github.com/replicatedhq/kotsadm v1.15.0-beta.2
	github.com/replicatedhq/troubleshoot v0.9.31
	github.com/replicatedhq/yaml/v3 v3.0.0-beta5-replicatedhq
	github.com/robfig/cron/v3 v3.0.0
	github.com/segmentio/ksuid v1.0.2
	github.com/sergi/go-diff v1.0.0
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.6.2
	github.com/stevvooe/resumable v0.0.0-20180830230917-22b14a53ba50 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/vmware-tanzu/velero v1.2.0
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	go.uber.org/zap v1.13.0
	go.undefinedlabs.com/scopeagent v0.1.15
	golang.org/x/crypto v0.0.0-20200414173820-0848c9571904
	google.golang.org/api v0.6.1-0.20190607001116-5213b8090861
	gopkg.in/go-playground/assert.v1 v1.2.1
	gopkg.in/ini.v1 v1.51.0
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cluster-bootstrap v0.17.3
	k8s.io/kubernetes v1.17.3
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/application v0.8.1
	sigs.k8s.io/controller-runtime v0.4.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65
	github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
	github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
	k8s.io/api => k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.3
	k8s.io/apiserver => k8s.io/apiserver v0.17.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.3
	k8s.io/client-go => k8s.io/client-go v0.17.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.3
	k8s.io/code-generator => k8s.io/code-generator v0.17.3
	k8s.io/component-base => k8s.io/component-base v0.17.3
	k8s.io/cri-api => k8s.io/cri-api v0.17.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.3
	k8s.io/gengo => k8s.io/gengo v0.0.0-20190822140433-26a664648505
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1
	k8s.io/klog => k8s.io/klog v1.0.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.3
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.3
	k8s.io/kubectl => k8s.io/kubectl v0.17.3
	k8s.io/kubelet => k8s.io/kubelet v0.17.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.3
	k8s.io/metrics => k8s.io/metrics v0.17.3
	k8s.io/node-api => k8s.io/node-api v0.17.3
	k8s.io/repo-infra => k8s.io/repo-infra v0.0.1-alpha.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.3
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.3
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.3
	k8s.io/utils => k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
)
