package upstream

import (
	"fmt"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_configureChart(t *testing.T) {
	origPodNamespace := util.PodNamespace
	util.PodNamespace = "test-namespace"
	defer func() {
		util.PodNamespace = origPodNamespace
	}()

	testReplicatedChartNames := []string{
		"replicated",
		"replicated-sdk",
	}

	type Test struct {
		name         string
		isAirgap     bool
		chartContent map[string]string
		want         map[string]string
		wantErr      bool
	}

	tests := []Test{
		{
			name:     "online - a standalone non-replicated chart",
			isAirgap: false,
			chartContent: map[string]string{
				"non-replicated/Chart.yaml": `apiVersion: v1
name: not-replicated
version: 1.0.0
description: Not a Replicated Chart
`,
				"non-replicated/values.yaml": `# this values.yaml file should not change

# do not change global values
global:
  some: value

# use this value to configure the chart
some: value
`,
			},
			want: map[string]string{
				"non-replicated/Chart.yaml": `apiVersion: v1
name: not-replicated
version: 1.0.0
description: Not a Replicated Chart
`,
				"non-replicated/values.yaml": `# this values.yaml file should not change

# do not change global values
global:
  some: value

# use this value to configure the chart
some: value
`,
			},
			wantErr: false,
		},
		{
			name:     "airgap - a standalone non-replicated chart",
			isAirgap: true,
			chartContent: map[string]string{
				"non-replicated/Chart.yaml": `apiVersion: v1
name: not-replicated
version: 1.0.0
description: Not a Replicated Chart
`,
				"non-replicated/values.yaml": `# this values.yaml file should not change

# do not change global values
global:
  some: value

# use this value to configure the chart
some: value
`,
			},
			want: map[string]string{
				"non-replicated/Chart.yaml": `apiVersion: v1
name: not-replicated
version: 1.0.0
description: Not a Replicated Chart
`,
				"non-replicated/values.yaml": `# this values.yaml file should not change

# do not change global values
global:
  some: value

# use this value to configure the chart
some: value
`,
			},
			wantErr: false,
		},
		{
			name:     "online - an nginx chart with the 'common' subchart only",
			isAirgap: false,
			chartContent: map[string]string{
				"nginx/Chart.yaml": `apiVersion: v2
name: nginx
version: 12.0.1
description: An NGINX Chart
`,
				"nginx/values.yaml": `## @section Global parameters
## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry, imagePullSecrets and storageClass

## @param global.imageRegistry Global Docker image registry
## @param global.imagePullSecrets Global Docker registry secret names as an array
##
global:
  imageRegistry: ""
  ## E.g.
  ## imagePullSecrets:
  ##   - myRegistryKeySecretName
  ##
  imagePullSecrets: []

## @section Common parameters

## @param nameOverride String to partially override nginx.fullname template (will maintain the release name)
##
nameOverride: ""
`,
				"nginx/charts/common/Chart.yaml": `apiVersion: v2
name: common
version: 1.13.1
description: A Common Chart
`,
				"nginx/charts/common/values.yaml": `# do not change this file

# do not change global values
global:
  some: value

# keep this comment
another: value
`,
			},
			want: map[string]string{
				"nginx/Chart.yaml": `apiVersion: v2
name: nginx
version: 12.0.1
description: An NGINX Chart
`,
				"nginx/values.yaml": `## @section Global parameters
## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry, imagePullSecrets and storageClass

## @param global.imageRegistry Global Docker image registry
## @param global.imagePullSecrets Global Docker registry secret names as an array
##
global:
  imageRegistry: ""
  ## E.g.
  ## imagePullSecrets:
  ##   - myRegistryKeySecretName
  ##
  imagePullSecrets: []

## @section Common parameters

## @param nameOverride String to partially override nginx.fullname template (will maintain the release name)
##
nameOverride: ""
`,
				"nginx/charts/common/Chart.yaml": `apiVersion: v2
name: common
version: 1.13.1
description: A Common Chart
`,
				"nginx/charts/common/values.yaml": `# do not change this file

# do not change global values
global:
  some: value

# keep this comment
another: value
`,
			},
			wantErr: false,
		},
		{
			name:     "airgap - an nginx chart with the 'common' subchart only",
			isAirgap: true,
			chartContent: map[string]string{
				"nginx/Chart.yaml": `apiVersion: v2
name: nginx
version: 12.0.1
description: An NGINX Chart
`,
				"nginx/values.yaml": `## @section Global parameters
## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry, imagePullSecrets and storageClass

## @param global.imageRegistry Global Docker image registry
## @param global.imagePullSecrets Global Docker registry secret names as an array
##
global:
  imageRegistry: ""
  ## E.g.
  ## imagePullSecrets:
  ##   - myRegistryKeySecretName
  ##
  imagePullSecrets: []

## @section Common parameters

## @param nameOverride String to partially override nginx.fullname template (will maintain the release name)
##
nameOverride: ""
`,
				"nginx/charts/common/Chart.yaml": `apiVersion: v2
name: common
version: 1.13.1
description: A Common Chart
`,
				"nginx/charts/common/values.yaml": `# do not change this file

# do not change global values
global:
  some: value

# keep this comment
another: value
`,
			},
			want: map[string]string{
				"nginx/Chart.yaml": `apiVersion: v2
name: nginx
version: 12.0.1
description: An NGINX Chart
`,
				"nginx/values.yaml": `## @section Global parameters
## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry, imagePullSecrets and storageClass

## @param global.imageRegistry Global Docker image registry
## @param global.imagePullSecrets Global Docker registry secret names as an array
##
global:
  imageRegistry: ""
  ## E.g.
  ## imagePullSecrets:
  ##   - myRegistryKeySecretName
  ##
  imagePullSecrets: []

## @section Common parameters

## @param nameOverride String to partially override nginx.fullname template (will maintain the release name)
##
nameOverride: ""
`,
				"nginx/charts/common/Chart.yaml": `apiVersion: v2
name: common
version: 1.13.1
description: A Common Chart
`,
				"nginx/charts/common/values.yaml": `# do not change this file

# do not change global values
global:
  some: value

# keep this comment
another: value
`,
			},
			wantErr: false,
		},
	}

	// Generate dynamic tests using the supported replicated chart names
	for _, chartName := range testReplicatedChartNames {
		tests = append(tests, Test{
			name:     "online - a standalone replicated chart",
			isAirgap: false,
			chartContent: map[string]string{
				"replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"replicated/values.yaml": `# preserve this comment

license: online-license
appName: online-app-name
channelID: online-channel-id
channelName: online-channel-name
channelSequence: 2
releaseCreatedAt: "2023-10-02T00:00:00Z"
releaseNotes: override my release notes
releaseSequence: 1
statusInformers:
  - deployment/replicated
  - service/replicated
versionLabel: 1.0.0
# and this comment

global:
  replicated:
    licenseID: online-license-id
    channelName: online-channel-name
    customerName: Online Customer Name
    customerEmail: online-customer@example.com
    licenseType: dev
    dockerconfigjson: bm90LWEtZG9ja2VyLWNvbmZpZy1qc29uCg==
    licenseFields:
      expires_at:
        name: expires_at
        title: Expiration
        description: License Expiration
        value: ""
        valueType: String
        signature:
          v1: nwZmD/sMFzKKxkd7JaAcKU/2uBE5m23w7+8xqLMXjUturMVCF5cF66EVMAibb2nHOqytie+N35GYSwIeTd16PKwbFBDd12c2E5M9COWwjVRcVTz4OnNWmHv9PEqZIbXhvfCLlyJ/aY3zV9Pno1VLFcYxGMrBugncEo4ecHkEbaVp3VLS4wn8EykAC1byvYBshzEXppYYd3c6a9cNw50Z6inI/IaKVxIForuz+Yn5uRAsjRyCY2auBCMeHMhY+CQ+4Vl5WtGjuJuE1g7t8AVZqt2JDBgDuxZAZX/JGncfzUaaDl87athMTtBKnFkTnCl34UXPkhsgM0LC4YoUiyKYjQ==
some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"replicated/values.yaml": `# preserve this comment

license: online-license
appName: online-app-name
channelID: online-channel-id
channelName: online-channel-name
channelSequence: 2
releaseCreatedAt: "2023-10-02T00:00:00Z"
releaseNotes: override my release notes
releaseSequence: 1
statusInformers:
  - deployment/replicated
  - service/replicated
versionLabel: 1.0.0
# and this comment

global:
  replicated:
    licenseID: online-license-id
    channelName: online-channel-name
    customerName: Online Customer Name
    customerEmail: online-customer@example.com
    licenseType: dev
    dockerconfigjson: bm90LWEtZG9ja2VyLWNvbmZpZy1qc29uCg==
    licenseFields:
      expires_at:
        name: expires_at
        title: Expiration
        description: License Expiration
        value: ""
        valueType: String
        signature:
          v1: nwZmD/sMFzKKxkd7JaAcKU/2uBE5m23w7+8xqLMXjUturMVCF5cF66EVMAibb2nHOqytie+N35GYSwIeTd16PKwbFBDd12c2E5M9COWwjVRcVTz4OnNWmHv9PEqZIbXhvfCLlyJ/aY3zV9Pno1VLFcYxGMrBugncEo4ecHkEbaVp3VLS4wn8EykAC1byvYBshzEXppYYd3c6a9cNw50Z6inI/IaKVxIForuz+Yn5uRAsjRyCY2auBCMeHMhY+CQ+4Vl5WtGjuJuE1g7t8AVZqt2JDBgDuxZAZX/JGncfzUaaDl87athMTtBKnFkTnCl34UXPkhsgM0LC4YoUiyKYjQ==
some: value
# and this comment as well

additionalMetricsEndpoint: http://kotsadm.test-namespace.svc.cluster.local:3000/api/v1/app/metrics
appID: app-id
isAirgap: false
replicatedID: kotsadm-id
userAgent: KOTS/v0.0.0-unknown
`,
			},
			wantErr: false,
		})

		tests = append(tests, Test{
			name:     "airgap - a standalone replicated chart",
			isAirgap: true,
			chartContent: map[string]string{
				"replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"replicated/values.yaml": `# preserve this comment

license: ""
appName: app-name
channelID: channel-id
channelName: channel-name
channelSequence: 2
releaseCreatedAt: "2023-10-02T00:00:00Z"
releaseNotes: override my release notes
releaseSequence: 1
statusInformers:
  - deployment/replicated
  - service/replicated
versionLabel: 1.0.0
# and this comment

some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"replicated/values.yaml": `# preserve this comment

license: |
  apiVersion: kots.io/v1beta1
  kind: License
  metadata:
    creationTimestamp: null
    name: kots-license
  spec:
    appSlug: app-slug
    channelName: channel-name
    customerEmail: customer@example.com
    customerName: Customer Name
    endpoint: https://replicated.app
    entitlements:
      license-field:
        description: This is a license field
        title: License Field
        value: license-field-value
        valueType: string
    licenseID: license-id
    licenseType: dev
    signature: ""
  status: {}
appName: app-name
channelID: channel-id
channelName: channel-name
channelSequence: 2
releaseCreatedAt: "2023-10-02T00:00:00Z"
releaseNotes: override my release notes
releaseSequence: 1
statusInformers:
  - deployment/replicated
  - service/replicated
versionLabel: 1.0.0
# and this comment

some: value
# and this comment as well

additionalMetricsEndpoint: http://kotsadm.test-namespace.svc.cluster.local:3000/api/v1/app/metrics
appID: app-id
isAirgap: true
replicatedID: kotsadm-id
userAgent: KOTS/v0.0.0-unknown
global:
  replicated:
    channelName: channel-name
    customerEmail: customer@example.com
    customerName: Customer Name
    dockerconfigjson: eyJhdXRocyI6eyJjdXN0b20ucHJveHkuY29tIjp7ImF1dGgiOiJiR2xqWlc1elpTMXBaRHBzYVdObGJuTmxMV2xrIn0sImN1c3RvbS5yZWdpc3RyeS5jb20iOnsiYXV0aCI6ImJHbGpaVzV6WlMxcFpEcHNhV05sYm5ObExXbGsifX19
    licenseFields:
      license-field:
        description: This is a license field
        name: license-field
        title: License Field
        value: license-field-value
        valueType: string
    licenseID: license-id
    licenseType: dev
`,
			},
			wantErr: false,
		})

		tests = append(tests, Test{
			name:     "online - a guestbook chart with the replicated subchart",
			isAirgap: false,
			chartContent: map[string]string{
				"guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"guestbook/values.yaml": fmt.Sprintf(`affinity: {}

# use this value to override the chart name
fullnameOverride: ""

# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
%s:
  license: online-license
  appName: online-app-name
  channelID: online-channel-id
  channelName: online-channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
global:
  replicated:
    licenseID: online-license-id
    channelName: online-channel-name
    customerName: Online Customer Name
    customerEmail: online-customer@example.com
    licenseType: dev
    dockerconfigjson: bm90LWEtZG9ja2VyLWNvbmZpZy1qc29uCg==
    licenseFields:
      expires_at:
        name: expires_at
        title: Expiration
        description: License Expiration
        value: ""
        valueType: String
        signature:
          v1: nwZmD/sMFzKKxkd7JaAcKU/2uBE5m23w7+8xqLMXjUturMVCF5cF66EVMAibb2nHOqytie+N35GYSwIeTd16PKwbFBDd12c2E5M9COWwjVRcVTz4OnNWmHv9PEqZIbXhvfCLlyJ/aY3zV9Pno1VLFcYxGMrBugncEo4ecHkEbaVp3VLS4wn8EykAC1byvYBshzEXppYYd3c6a9cNw50Z6inI/IaKVxIForuz+Yn5uRAsjRyCY2auBCMeHMhY+CQ+4Vl5WtGjuJuE1g7t8AVZqt2JDBgDuxZAZX/JGncfzUaaDl87athMTtBKnFkTnCl34UXPkhsgM0LC4YoUiyKYjQ==
`, chartName),
				"guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"guestbook/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"guestbook/values.yaml": fmt.Sprintf(`affinity: {}
# use this value to override the chart name
fullnameOverride: ""
# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
%s:
  license: online-license
  appName: online-app-name
  channelID: online-channel-id
  channelName: online-channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
  additionalMetricsEndpoint: http://kotsadm.test-namespace.svc.cluster.local:3000/api/v1/app/metrics
  appID: app-id
  isAirgap: false
  replicatedID: kotsadm-id
  userAgent: KOTS/v0.0.0-unknown
global:
  replicated:
    licenseID: online-license-id
    channelName: online-channel-name
    customerName: Online Customer Name
    customerEmail: online-customer@example.com
    licenseType: dev
    dockerconfigjson: bm90LWEtZG9ja2VyLWNvbmZpZy1qc29uCg==
    licenseFields:
      expires_at:
        name: expires_at
        title: Expiration
        description: License Expiration
        value: ""
        valueType: String
        signature:
          v1: nwZmD/sMFzKKxkd7JaAcKU/2uBE5m23w7+8xqLMXjUturMVCF5cF66EVMAibb2nHOqytie+N35GYSwIeTd16PKwbFBDd12c2E5M9COWwjVRcVTz4OnNWmHv9PEqZIbXhvfCLlyJ/aY3zV9Pno1VLFcYxGMrBugncEo4ecHkEbaVp3VLS4wn8EykAC1byvYBshzEXppYYd3c6a9cNw50Z6inI/IaKVxIForuz+Yn5uRAsjRyCY2auBCMeHMhY+CQ+4Vl5WtGjuJuE1g7t8AVZqt2JDBgDuxZAZX/JGncfzUaaDl87athMTtBKnFkTnCl34UXPkhsgM0LC4YoUiyKYjQ==
`, chartName),
				"guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"guestbook/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			wantErr: false,
		})

		tests = append(tests, Test{
			name:     "airgap - a guestbook chart with the replicated subchart",
			isAirgap: true,
			chartContent: map[string]string{
				"guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"guestbook/values.yaml": fmt.Sprintf(`affinity: {}

# use this value to override the chart name
fullnameOverride: ""

# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
%s:
  appName: app-name
  channelID: channel-id
  channelName: channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
`, chartName),
				"guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"guestbook/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"guestbook/values.yaml": fmt.Sprintf(`affinity: {}
# use this value to override the chart name
fullnameOverride: ""
# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
%s:
  appName: app-name
  channelID: channel-id
  channelName: channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
  additionalMetricsEndpoint: http://kotsadm.test-namespace.svc.cluster.local:3000/api/v1/app/metrics
  appID: app-id
  isAirgap: true
  license: |
    apiVersion: kots.io/v1beta1
    kind: License
    metadata:
      creationTimestamp: null
      name: kots-license
    spec:
      appSlug: app-slug
      channelName: channel-name
      customerEmail: customer@example.com
      customerName: Customer Name
      endpoint: https://replicated.app
      entitlements:
        license-field:
          description: This is a license field
          title: License Field
          value: license-field-value
          valueType: string
      licenseID: license-id
      licenseType: dev
      signature: ""
    status: {}
  replicatedID: kotsadm-id
  userAgent: KOTS/v0.0.0-unknown
global:
  replicated:
    channelName: channel-name
    customerEmail: customer@example.com
    customerName: Customer Name
    dockerconfigjson: eyJhdXRocyI6eyJjdXN0b20ucHJveHkuY29tIjp7ImF1dGgiOiJiR2xqWlc1elpTMXBaRHBzYVdObGJuTmxMV2xrIn0sImN1c3RvbS5yZWdpc3RyeS5jb20iOnsiYXV0aCI6ImJHbGpaVzV6WlMxcFpEcHNhV05sYm5ObExXbGsifX19
    licenseFields:
      license-field:
        description: This is a license field
        name: license-field
        title: License Field
        value: license-field-value
        valueType: string
    licenseID: license-id
    licenseType: dev
`, chartName),
				"guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"guestbook/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			wantErr: false,
		})

		tests = append(tests, Test{
			name:     "online - a redis chart with the replicated subchart and predefined replicated and global values",
			isAirgap: false,
			chartContent: map[string]string{
				"redis/Chart.yaml": `apiVersion: v1
name: redis
version: 5.0.7
description: A Redis Chart
`,
				"redis/values.yaml": fmt.Sprintf(`## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry and imagePullSecrets
##
global:
  # imageRegistry: myRegistryName
  # imagePullSecrets:
  #   - myRegistryKeySecretName
  # storageClass: myStorageClass
  redis: {}
  replicated:
    some: value
    licenseID: online-license-id
    channelName: online-channel-name
    customerName: Online Customer Name
    customerEmail: online-customer@example.com
    licenseType: dev
    dockerconfigjson: bm90LWEtZG9ja2VyLWNvbmZpZy1qc29uCg==
    licenseFields:
      expires_at:
        name: expires_at
        title: Expiration
        description: License Expiration
        value: ""
        valueType: String
        signature:
          v1: nwZmD/sMFzKKxkd7JaAcKU/2uBE5m23w7+8xqLMXjUturMVCF5cF66EVMAibb2nHOqytie+N35GYSwIeTd16PKwbFBDd12c2E5M9COWwjVRcVTz4OnNWmHv9PEqZIbXhvfCLlyJ/aY3zV9Pno1VLFcYxGMrBugncEo4ecHkEbaVp3VLS4wn8EykAC1byvYBshzEXppYYd3c6a9cNw50Z6inI/IaKVxIForuz+Yn5uRAsjRyCY2auBCMeHMhY+CQ+4Vl5WtGjuJuE1g7t8AVZqt2JDBgDuxZAZX/JGncfzUaaDl87athMTtBKnFkTnCl34UXPkhsgM0LC4YoUiyKYjQ==

# values related to the replicated subchart
%s:
  some: value
  license: online-license
  appName: online-app-name
  channelID: online-channel-id
  channelName: online-channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
`, chartName),
				"redis/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"redis/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"redis/Chart.yaml": `apiVersion: v1
name: redis
version: 5.0.7
description: A Redis Chart
`,
				"redis/values.yaml": fmt.Sprintf(`## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry and imagePullSecrets
##
global:
  # imageRegistry: myRegistryName
  # imagePullSecrets:
  #   - myRegistryKeySecretName
  # storageClass: myStorageClass
  redis: {}
  replicated:
    some: value
    licenseID: online-license-id
    channelName: online-channel-name
    customerName: Online Customer Name
    customerEmail: online-customer@example.com
    licenseType: dev
    dockerconfigjson: bm90LWEtZG9ja2VyLWNvbmZpZy1qc29uCg==
    licenseFields:
      expires_at:
        name: expires_at
        title: Expiration
        description: License Expiration
        value: ""
        valueType: String
        signature:
          v1: nwZmD/sMFzKKxkd7JaAcKU/2uBE5m23w7+8xqLMXjUturMVCF5cF66EVMAibb2nHOqytie+N35GYSwIeTd16PKwbFBDd12c2E5M9COWwjVRcVTz4OnNWmHv9PEqZIbXhvfCLlyJ/aY3zV9Pno1VLFcYxGMrBugncEo4ecHkEbaVp3VLS4wn8EykAC1byvYBshzEXppYYd3c6a9cNw50Z6inI/IaKVxIForuz+Yn5uRAsjRyCY2auBCMeHMhY+CQ+4Vl5WtGjuJuE1g7t8AVZqt2JDBgDuxZAZX/JGncfzUaaDl87athMTtBKnFkTnCl34UXPkhsgM0LC4YoUiyKYjQ==
# values related to the replicated subchart
%s:
  some: value
  license: online-license
  appName: online-app-name
  channelID: online-channel-id
  channelName: online-channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
  additionalMetricsEndpoint: http://kotsadm.test-namespace.svc.cluster.local:3000/api/v1/app/metrics
  appID: app-id
  isAirgap: false
  replicatedID: kotsadm-id
  userAgent: KOTS/v0.0.0-unknown
`, chartName),
				"redis/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"redis/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			wantErr: false,
		})

		tests = append(tests, Test{
			name:     "airgap - a redis chart with the replicated subchart and predefined replicated and global values",
			isAirgap: true,
			chartContent: map[string]string{
				"redis/Chart.yaml": `apiVersion: v1
name: redis
version: 5.0.7
description: A Redis Chart
`,
				"redis/values.yaml": fmt.Sprintf(`## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry and imagePullSecrets
##
global:
  # imageRegistry: myRegistryName
  # imagePullSecrets:
  #   - myRegistryKeySecretName
  # storageClass: myStorageClass
  redis: {}
  replicated:
    some: value

# values related to the replicated subchart
%s:
  some: value
  appName: app-name
  channelID: channel-id
  channelName: channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
`, chartName),
				"redis/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"redis/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"redis/Chart.yaml": `apiVersion: v1
name: redis
version: 5.0.7
description: A Redis Chart
`,
				"redis/values.yaml": fmt.Sprintf(`## Global Docker image parameters
## Please, note that this will override the image parameters, including dependencies, configured to use the global value
## Current available global Docker image parameters: imageRegistry and imagePullSecrets
##
global:
  # imageRegistry: myRegistryName
  # imagePullSecrets:
  #   - myRegistryKeySecretName
  # storageClass: myStorageClass
  redis: {}
  replicated:
    some: value
    channelName: channel-name
    customerEmail: customer@example.com
    customerName: Customer Name
    dockerconfigjson: eyJhdXRocyI6eyJjdXN0b20ucHJveHkuY29tIjp7ImF1dGgiOiJiR2xqWlc1elpTMXBaRHBzYVdObGJuTmxMV2xrIn0sImN1c3RvbS5yZWdpc3RyeS5jb20iOnsiYXV0aCI6ImJHbGpaVzV6WlMxcFpEcHNhV05sYm5ObExXbGsifX19
    licenseFields:
      license-field:
        description: This is a license field
        name: license-field
        title: License Field
        value: license-field-value
        valueType: string
    licenseID: license-id
    licenseType: dev
# values related to the replicated subchart
%s:
  some: value
  appName: app-name
  channelID: channel-id
  channelName: channel-name
  channelSequence: 2
  releaseCreatedAt: "2023-10-02T00:00:00Z"
  releaseNotes: override my release notes
  releaseSequence: 1
  statusInformers:
    - deployment/replicated
    - service/replicated
  versionLabel: 1.0.0
  additionalMetricsEndpoint: http://kotsadm.test-namespace.svc.cluster.local:3000/api/v1/app/metrics
  appID: app-id
  isAirgap: true
  license: |
    apiVersion: kots.io/v1beta1
    kind: License
    metadata:
      creationTimestamp: null
      name: kots-license
    spec:
      appSlug: app-slug
      channelName: channel-name
      customerEmail: customer@example.com
      customerName: Customer Name
      endpoint: https://replicated.app
      entitlements:
        license-field:
          description: This is a license field
          title: License Field
          value: license-field-value
          valueType: string
      licenseID: license-id
      licenseType: dev
      signature: ""
    status: {}
  replicatedID: kotsadm-id
  userAgent: KOTS/v0.0.0-unknown
`, chartName),
				"redis/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"redis/charts/replicated/values.yaml": `# preserve this comment

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			wantErr: false,
		})

		tests = append(tests, Test{
			name:     "online - a postgresql chart with replicated as subsubchart",
			isAirgap: false,
			chartContent: map[string]string{
				"postgresql/Chart.yaml": `apiVersion: v2
name: postgresql
version: 11.6.0
description: A Postgresql Chart
`,
				"postgresql/values.yaml": `extraEnv: []

# override global values here
global:
  postgresql: {}

# additional values can be added here
`,
				"postgresql/charts/guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"postgresql/charts/guestbook/values.yaml": `affinity: {}

# use this value to override the chart name
fullnameOverride: ""

# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
`,
				"postgresql/charts/guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"postgresql/charts/guestbook/charts/replicated/values.yaml": `# this file should NOT change

# global values should NOT be updated
global:
  keep: this-value

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"postgresql/Chart.yaml": `apiVersion: v2
name: postgresql
version: 11.6.0
description: A Postgresql Chart
`,
				"postgresql/values.yaml": `extraEnv: []

# override global values here
global:
  postgresql: {}

# additional values can be added here
`,
				"postgresql/charts/guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"postgresql/charts/guestbook/values.yaml": `affinity: {}

# use this value to override the chart name
fullnameOverride: ""

# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
`,
				"postgresql/charts/guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"postgresql/charts/guestbook/charts/replicated/values.yaml": `# this file should NOT change

# global values should NOT be updated
global:
  keep: this-value

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
		})

		tests = append(tests, Test{
			name:     "airgap - a postgresql chart with replicated as subsubchart",
			isAirgap: true,
			chartContent: map[string]string{
				"postgresql/Chart.yaml": `apiVersion: v2
name: postgresql
version: 11.6.0
description: A Postgresql Chart
`,
				"postgresql/values.yaml": `extraEnv: []

# override global values here
global:
  postgresql: {}

# additional values can be added here
`,
				"postgresql/charts/guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"postgresql/charts/guestbook/values.yaml": `affinity: {}

# use this value to override the chart name
fullnameOverride: ""

# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
`,
				"postgresql/charts/guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"postgresql/charts/guestbook/charts/replicated/values.yaml": `# this file should NOT change

# global values should NOT be updated
global:
  keep: this-value

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
			want: map[string]string{
				"postgresql/Chart.yaml": `apiVersion: v2
name: postgresql
version: 11.6.0
description: A Postgresql Chart
`,
				"postgresql/values.yaml": `extraEnv: []

# override global values here
global:
  postgresql: {}

# additional values can be added here
`,
				"postgresql/charts/guestbook/Chart.yaml": `apiVersion: v2
name: guestbook
version: 1.16.0
description: A Guestbook Chart
`,
				"postgresql/charts/guestbook/values.yaml": `affinity: {}

# use this value to override the chart name
fullnameOverride: ""

# use this value to set the image pull policy
image:
  pullPolicy: IfNotPresent
`,
				"postgresql/charts/guestbook/charts/replicated/Chart.yaml": fmt.Sprintf(`apiVersion: v1
name: %s
version: 1.0.0
description: A Replicated Chart
`, chartName),
				"postgresql/charts/guestbook/charts/replicated/values.yaml": `# this file should NOT change

# global values should NOT be updated
global:
  keep: this-value

channelName: keep-this-channel-name
# and this comment

some: value
# and this comment as well
`,
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chartBytes, err := util.FilesToTGZ(tt.chartContent)
			require.NoError(t, err)

			upstream := &types.Upstream{
				License: &kotsv1beta1.License{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "License",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "kots-license",
					},
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID:   "license-id",
						AppSlug:     "app-slug",
						ChannelName: "channel-name",
						Endpoint:    "https://replicated.app",
						Entitlements: map[string]kotsv1beta1.EntitlementField{
							"license-field": {
								Title:       "License Field",
								Description: "This is a license field",
								ValueType:   "string",
								Value: kotsv1beta1.EntitlementValue{
									Type:   kotsv1beta1.String,
									StrVal: "license-field-value",
								},
							},
						},
						CustomerEmail: "customer@example.com",
						CustomerName:  "Customer Name",
						LicenseType:   "dev",
						Signature:     []byte{},
					},
				},
				ReplicatedRegistryDomain: "custom.registry.com",
				ReplicatedProxyDomain:    "custom.proxy.com",
				ReplicatedChartNames:     testReplicatedChartNames,
			}

			writeOptions := types.WriteOptions{
				KotsadmID: "kotsadm-id",
				AppID:     "app-id",
				IsAirgap:  tt.isAirgap,
			}

			got, err := configureChart(chartBytes, upstream, writeOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("configureChart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotFiles, err := util.TGZToFiles(got)
			require.NoError(t, err)

			for filename, wantContent := range tt.want {
				gotContent := gotFiles[filename]
				if gotContent != wantContent {
					t.Errorf("configureChart() %s: %v", filename, diffString(gotContent, wantContent))
				}
			}
		})
	}
}

func diffString(got, want string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(got),
		B:        difflib.SplitLines(want),
		FromFile: "Got",
		ToFile:   "Want",
		Context:  1,
	}
	diffStr, _ := difflib.GetUnifiedDiffString(diff)
	return fmt.Sprintf("got:\n%s \n\nwant:\n%s \n\ndiff:\n%s", got, want, diffStr)
}
