import React from "react";
import Modal from "react-modal";
import MonacoEditor from "@monaco-editor/react";

export default function HelmValuesModal({
  showHelmValuesModal,
  hideHelmValuesModal,
}) {

  return (
    <Modal
      isOpen={showHelmValuesModal}
      onRequestClose={hideHelmValuesModal}
      shouldReturnFocusAfterClose={false}
      contentLabel=""
      ariaHideApp={false}
      className="Modal LargeSize"
    >
      <div className="Modal-header has-border flex">
        <h3 className="flex1">Helm Values</h3>
        <button className="secondary blue btn u-marginRight--10" onClick={() => {}}>Download</button>
        <span className="icon u-grayX-icon u-cursor--pointer"></span>
      </div>
      <div className="Modal-body">
        <MonacoEditor
          ref={(editor) => {
            this.monacoEditor = editor;
          }}
          height="420px"
          language={"yaml"}
          value={`
# SPDX-FileCopyrightText: the secureCodeBox authors
#
# SPDX-License-Identifier: Apache-2.0

# Default values for http-webhook.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.

replicaCount: 1

image:
  # image.repository -- Container Image
  repository: docker.io/mendhak/http-https-echo
  # image.tag -- The image tag
  # @default -- defaults to the latest version because the appVersion tag is not available at docker.io
  tag: latest
  # -- Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
  pullPolicy: IfNotPresent

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

# -- add labels to the deployment, service and pods
labels: {}

# -- add annotations to the deployment, service and pods
annotations: {}

serviceAccount:
  # Specifies whether a service account should be created
  create: true
  # Annotations to add to the service account
  annotations: {}
  # The name of the service account to use.
  # If not set and create is true, a name is generated using the fullname template
  name: ""

# -- deprecated. use \`labels\` instead. Will be removed in v3. todo(@J12934) remove podAnnotations in v3
podAnnotations: {}

podSecurityContext: {}
  # fsGroup: 2000

securityContext: {}
  # capabilities:
  #   drop:
  #   - ALL
  # readOnlyRootFilesystem: true
  # runAsNonRoot: true
  # runAsUser: 1000

service:
  type: ClusterIP
  port: 80

ingress:
  enabled: false
  annotations: {}
    # kubernetes.io/ingress.class: nginx
    # kubernetes.io/tls-acme: "true"
  hosts:
    - host: chart-example.local
      paths: []
  tls: []
  #  - secretName: chart-example-tls
  #    hosts:
  #      - chart-example.local

resources: {}
  # We usually recommend not to specify default resources and to leave this as a conscious
  # choice for the user. This also increases chances charts run on environments with little
  # resources, such as Minikube. If you do want to specify resources, uncomment the following
  # lines, adjust them as necessary, and remove the curly braces after 'resources:'.
  # limits:
  #   cpu: 100m
  #   memory: 128Mi
  # requests:
  #   cpu: 100m
  #   memory: 128Mi

autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 100
  targetCPUUtilizationPercentage: 80
  # targetMemoryUtilizationPercentage: 80

nodeSelector: {}

tolerations: []

affinity: {}

}`}
              />
      </div>
    </Modal>
  );
}

export { HelmValuesModal }