#!/usr/bin/env bash

set -euo pipefail

: "${K0S_VERSION:=v1.30.14+k0s.0}"
: "${KOTS_OLD_VERSION:?KOTS_OLD_VERSION is required}"
: "${IS_UPGRADE:?IS_UPGRADE is required}"
: "${IS_AIRGAPPED:?IS_AIRGAPPED is required}"

readonly REGISTRY_USERNAME="kotsadm"
readonly REGISTRY_PASSWORD="password"
readonly REGISTRY_PORT="30443"
readonly AIRGAP_ASSET_DIR="/opt/kots-regression-airgap"
readonly LOCAL_PATH_PROVISIONER_VERSION="v0.0.37"
readonly KUBE_PROMETHEUS_VERSION="v0.14.0"

wait_for_node() {
  for _ in $(seq 1 60); do
    if k0s kubectl get nodes --no-headers 2>/dev/null | awk '$2 == "Ready" { found=1 } END { exit !found }'; then
      return
    fi
    sleep 5
  done

  k0s status || true
  k0s kubectl get nodes -o wide || true
  return 1
}

install_dependencies() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y apache2-utils ca-certificates curl jq skopeo

  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y build-essential nodejs

  curl -sL https://github.com/replicatedhq/troubleshoot/releases/latest/download/support-bundle_linux_amd64.tar.gz |
    tar xz -C /usr/local/bin support-bundle
  chmod +x /usr/local/bin/support-bundle
  curl -sL https://raw.githubusercontent.com/replicatedhq/troubleshoot-specs/main/host/default.yaml \
    -o /tmp/support-bundle-host-spec.yaml
  curl -sL https://raw.githubusercontent.com/replicatedhq/troubleshoot-specs/main/in-cluster/default.yaml \
    -o /tmp/support-bundle-in-cluster-spec.yaml
}

install_k0s() {
  curl -fsSL \
    "https://github.com/k0sproject/k0s/releases/download/${K0S_VERSION}/k0s-${K0S_VERSION}-amd64" \
    -o /usr/local/bin/k0s
  chmod +x /usr/local/bin/k0s
  k0s install controller --single
  k0s start
  wait_for_node

  cat <<'EOF' > /usr/local/bin/kubectl
#!/bin/sh
exec k0s kubectl "$@"
EOF
  chmod +x /usr/local/bin/kubectl

  mkdir -p /root/.kube
  cp /var/lib/k0s/pki/admin.conf /root/.kube/config
  chmod 600 /root/.kube/config
  export KUBECONFIG=/root/.kube/config
}

install_storage() {
  curl -fsSL \
    "https://raw.githubusercontent.com/rancher/local-path-provisioner/${LOCAL_PATH_PROVISIONER_VERSION}/deploy/local-path-storage.yaml" \
    -o /tmp/local-path-storage.yaml
  sed -i 's|image: docker.io/library/busybox$|image: docker.io/library/busybox:1.36.1|' \
    /tmp/local-path-storage.yaml
  kubectl apply -f /tmp/local-path-storage.yaml
  kubectl annotate storageclass local-path storageclass.kubernetes.io/is-default-class=true --overwrite
  kubectl -n local-path-storage rollout status deployment/local-path-provisioner --timeout=2m

  # Exercise dynamic provisioning while the VM is online. This both validates
  # the StorageClass and caches the helper image needed after air-gapping.
  cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: local-path-smoke-test
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 16Mi
---
apiVersion: v1
kind: Pod
metadata:
  name: local-path-smoke-test
spec:
  restartPolicy: Never
  containers:
    - name: busybox
      image: busybox:1.36.1
      command: ["sh", "-c", "echo ready > /data/status && sleep 300"]
      volumeMounts:
        - name: data
          mountPath: /data
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: local-path-smoke-test
EOF
  kubectl wait --for=condition=Ready pod/local-path-smoke-test --timeout=2m
  kubectl delete pod/local-path-smoke-test pvc/local-path-smoke-test --wait=true
}

install_monitoring() {
  curl -fsSL \
    "https://github.com/prometheus-operator/kube-prometheus/archive/refs/tags/${KUBE_PROMETHEUS_VERSION}.tar.gz" \
    | tar xz -C /tmp

  local manifest_dir="/tmp/kube-prometheus-${KUBE_PROMETHEUS_VERSION#v}/manifests"
  kubectl apply --server-side -f "${manifest_dir}/setup"
  kubectl wait --for=condition=Established --all customresourcedefinition --timeout=2m
  kubectl apply -f "$manifest_dir"
  kubectl -n monitoring rollout status deployment/prometheus-operator --timeout=3m
  kubectl -n monitoring rollout status daemonset/node-exporter --timeout=3m
  for _ in $(seq 1 60); do
    if kubectl -n monitoring get statefulset/prometheus-k8s >/dev/null 2>&1; then
      kubectl -n monitoring rollout status statefulset/prometheus-k8s --timeout=5m
      return
    fi
    sleep 2
  done
  echo "Prometheus StatefulSet was not created" >&2
  return 1
}

install_registry() {
  local node_ip
  node_ip="$(hostname -I | awk '{print $1}')"
  printf '%s\n' "$node_ip" > /tmp/cmx-private-ip

  mkdir -p /tmp/kots-registry /var/lib/kots-registry
  openssl req -x509 -newkey rsa:2048 -nodes -days 2 \
    -keyout /tmp/kots-registry/tls.key \
    -out /tmp/kots-registry/tls.crt \
    -subj "/CN=${node_ip}" \
    -addext "subjectAltName=IP:${node_ip}"
  htpasswd -Bbn "$REGISTRY_USERNAME" "$REGISTRY_PASSWORD" > /tmp/kots-registry/htpasswd

  cp /tmp/kots-registry/tls.crt /usr/local/share/ca-certificates/kots-registry.crt
  update-ca-certificates

  k0s kubectl create namespace kots-registry
  k0s kubectl -n kots-registry create secret tls registry-tls \
    --cert=/tmp/kots-registry/tls.crt \
    --key=/tmp/kots-registry/tls.key
  k0s kubectl -n kots-registry create secret generic registry-auth \
    --from-file=htpasswd=/tmp/kots-registry/htpasswd

  cat <<EOF | k0s kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
  namespace: kots-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      labels:
        app: registry
    spec:
      containers:
        - name: registry
          image: registry:2
          ports:
            - name: https
              containerPort: 5000
              hostPort: 443
          env:
            - name: REGISTRY_HTTP_ADDR
              value: ":5000"
            - name: REGISTRY_HTTP_TLS_CERTIFICATE
              value: /certs/tls.crt
            - name: REGISTRY_HTTP_TLS_KEY
              value: /certs/tls.key
            - name: REGISTRY_AUTH
              value: htpasswd
            - name: REGISTRY_AUTH_HTPASSWD_REALM
              value: Registry
            - name: REGISTRY_AUTH_HTPASSWD_PATH
              value: /auth/htpasswd
          volumeMounts:
            - name: tls
              mountPath: /certs
              readOnly: true
            - name: auth
              mountPath: /auth
              readOnly: true
            - name: data
              mountPath: /var/lib/registry
      volumes:
        - name: tls
          secret:
            secretName: registry-tls
        - name: auth
          secret:
            secretName: registry-auth
        - name: data
          hostPath:
            path: /var/lib/kots-registry
            type: Directory
---
apiVersion: v1
kind: Service
metadata:
  name: registry
  namespace: kots-registry
spec:
  type: NodePort
  selector:
    app: registry
  ports:
    - name: https
      port: 443
      targetPort: 5000
      nodePort: ${REGISTRY_PORT}
EOF

  k0s kubectl -n kots-registry rollout status deployment/registry --timeout=2m

  # containerd reads the system CA pool when it starts.
  systemctl restart k0scontroller
  wait_for_node

  k0s kubectl create secret docker-registry registry-creds \
    --docker-server="${node_ip}" \
    --docker-username="$REGISTRY_USERNAME" \
    --docker-password="$REGISTRY_PASSWORD"
}

prepare_kots_binaries() {
  chmod +x /tmp/kots-nightly

  if [[ "$IS_UPGRADE" == "1" ]]; then
    curl -fsSL \
      "https://github.com/replicatedhq/kots/releases/download/${KOTS_OLD_VERSION}/kots_linux_amd64.tar.gz" |
      tar xz -C /tmp
    mv /tmp/kots /usr/local/bin/kubectl-kots
  else
    cp /tmp/kots-nightly /usr/local/bin/kubectl-kots
  fi
  chmod +x /usr/local/bin/kubectl-kots

  if [[ "$IS_AIRGAPPED" == "1" && "$IS_UPGRADE" == "1" ]]; then
    curl -fsSL \
      "https://github.com/replicatedhq/kots/releases/download/${KOTS_OLD_VERSION}/kotsadm.tar.gz" \
      -o /tmp/initial-kotsadm.tar.gz
  fi
}

prepare_playwright() {
  cd /home/cmx/playwright
  npm ci
  PLAYWRIGHT_BROWSERS_PATH=/home/cmx/playwright/.cache/ms-playwright \
    npx playwright install --with-deps chromium
  chown -R cmx:cmx /home/cmx/playwright
}

preload_airgap_assets() {
  if [[ "$IS_AIRGAPPED" != "1" ]]; then
    return 0
  fi

  mkdir -p "$AIRGAP_ASSET_DIR"
  cd /home/cmx/playwright

  case "${TEST_PATH}" in
    regression/@existing-airgapped-install-admin)
      customer_id="1riYWlpWEbuVSARQtZmZWZOv2TN"
      portal_auth="SFJZQVlF"
      sequences=(17 18 13 14)
      destinations=(initial-small-app-release.airgap update-small-app-release.airgap initial-app-release.airgap new-app-release.airgap)
      ;;
    regression/@existing-airgapped-install-minimal)
      customer_id="1vsAXZYrhkYhr0AMbYL8gmPAAJu"
      portal_auth="SkpWSzlG"
      sequences=(13 14 8 9)
      destinations=(initial-small-app-release.airgap update-small-app-release.airgap initial-app-release.airgap new-app-release.airgap)
      ;;
    regression/@existing-airgapped-upgrade-admin)
      customer_id="1tUlnuE56n4IMxXB48NsJdsIcfG"
      portal_auth="RjlQNlZW"
      sequences=(16 17)
      destinations=(initial-app-release.airgap new-app-release.airgap)
      ;;
    regression/@existing-airgapped-upgrade-minimal)
      customer_id="1wEWBiIJBCoenaCc9RUNuvvcpAP"
      portal_auth="MjlWVkYy"
      sequences=(8 9)
      destinations=(initial-app-release.airgap new-app-release.airgap)
      ;;
    *)
      echo "Unknown air-gap test path: ${TEST_PATH}" >&2
      return 1
      ;;
  esac

  for i in "${!sequences[@]}"; do
    bundle_url="$(curl -fsSL \
      "https://api.replicated.com/market/v3/airgap/images/url?customer_id=${customer_id}&channel_sequence=${sequences[$i]}" \
      -H "Authorization: Basic ${portal_auth}" | jq -r .url)"
    curl -fsSL "$bundle_url" -o "${destinations[$i]}"
  done

  readonly velero_version="v1.12.1"
  readonly velero_plugin_version="v1.8.1"
  curl -fsSL \
    "https://github.com/vmware-tanzu/velero/releases/download/${velero_version}/velero-${velero_version}-linux-amd64.tar.gz" \
    -o "velero-${velero_version}-linux-amd64.tar.gz"
  skopeo copy "docker://velero/velero:${velero_version}" \
    "oci-archive:${AIRGAP_ASSET_DIR}/velero.tar"
  skopeo copy "docker://velero/velero-plugin-for-aws:${velero_plugin_version}" \
    "oci-archive:${AIRGAP_ASSET_DIR}/velero-plugin-for-aws.tar"
  skopeo copy "docker://velero/velero-restore-helper:${velero_version}" \
    "oci-archive:${AIRGAP_ASSET_DIR}/velero-restore-helper.tar"

  chown -R cmx:cmx /home/cmx/playwright "$AIRGAP_ASSET_DIR"
}

main() {
  install_dependencies
  install_k0s
  install_storage
  install_monitoring
  install_registry
  prepare_kots_binaries
  prepare_playwright
  preload_airgap_assets
}

main "$@"
