#!/bin/bash

# This script is used to re-generate the join token for the kURL cluster.
# It utilizes the running Pod's service account to authenticate with the Kubernetes API server.

set -e

# Set cluster name and service account for kubeconfig
CLUSTERNAME=kubernetes
USERNAME=kubernetes

# Point to the internal API server hostname
APISERVER=https://kubernetes.default.svc

# Path to ServiceAccount
SERVICEACCOUNTPATH=/var/run/secrets/kubernetes.io/serviceaccount

# Read this Pod's namespace
NAMESPACE=$(cat ${SERVICEACCOUNTPATH}/namespace)

# Read the ServiceAccount bearer token
TOKEN=$(cat ${SERVICEACCOUNTPATH}/token)

# Read the base64 encoded CA cert
CACERT=$(base64 -w 0 ${SERVICEACCOUNTPATH}/ca.crt)

# Create the kubeconfig file
KUBECONFIG=/tmp/kubeconfig.yaml

cat << EOF >> $KUBECONFIG
apiVersion: v1
kind: Config
clusters:
  - name: ${CLUSTERNAME}
    cluster:
      certificate-authority-data: ${CACERT}
      server: ${APISERVER}
contexts:
  - name: ${USERNAME}@${CLUSTERNAME}
    context:
      cluster: ${CLUSTERNAME}
      namespace: ${NAMESPACE}
      user: ${USERNAME}
users:
  - name: ${USERNAME}
    user:
      token: ${TOKEN}
current-context: ${USERNAME}@${CLUSTERNAME}
EOF

# Regenerate the certs using the new kubeconfig
/usr/bin/kubeadm init phase upload-certs --upload-certs --kubeconfig $KUBECONFIG
