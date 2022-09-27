#!/usr/bin/env bash

function kotsadm() {
    local src="$DIR/addons/kotsadm/__KOTSADM_DIR__"
    local dst="$DIR/kustomize/kotsadm"

    validate_object_storage

    kotsadm_rename_postgres_pvc_1-12-2 "$src"

    cp "$src/kustomization.yaml" "$dst/"
    cp "$src/postgres.yaml" "$dst/"
    cp "$src/kotsadm.yaml" "$dst/"

    if kubernetes_resource_exists default statefulset kotsadm; then
        # reverse migration is not supported
        KOTSADM_DISABLE_S3="1"
    fi

    # Migrate kotsadm deployment to statefulset
    if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
        cp "$DIR/addons/kotsadm/__KOTSADM_DIR__/statefulset/kotsadm-statefulset.yaml" "$DIR/kustomize/kotsadm/kotsadm-statefulset.yaml"
        insert_resources "$dst/kustomization.yaml" kotsadm-statefulset.yaml
        # kotsadm v1.47+ does not use an object store for the archives, patch the migrate-s3 init container to migrate the data.
        # the migration process is intelligent enough to detect whether an object store and a bucket exists or not.
        kotsadm_api_patch_s3_migration
    else
        cp "$DIR/addons/kotsadm/__KOTSADM_DIR__/deployment/kotsadm-deployment.yaml" "$DIR/kustomize/kotsadm/kotsadm-deployment.yaml"
        insert_resources "$dst/kustomization.yaml" kotsadm-deployment.yaml
    fi

    kotsadm_secret_cluster_token
    kotsadm_secret_authstring
    kotsadm_secret_password
    kotsadm_secret_postgres
    kotsadm_secret_dex_postgres
    kotsadm_secret_s3           # this secret is only used for (re)configuring internal snapshots; will not be created if there is no object store 
    kotsadm_secret_session
    kotsadm_api_encryption_key

    if [ -n "$PROMETHEUS_VERSION" ]; then
        kotsadm_api_patch_prometheus
    fi

    if [ -n "$PROXY_ADDRESS" ]; then
        KUBERNETES_CLUSTER_IP=$(kubectl get services kubernetes --no-headers | awk '{ print $3 }')
        if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
            render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/statefulset/tmpl-kotsadm-proxy.yaml" > "$DIR/kustomize/kotsadm/kotsadm-proxy.yaml"
        else
            render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/deployment/tmpl-kotsadm-proxy.yaml" > "$DIR/kustomize/kotsadm/kotsadm-proxy.yaml"
        fi
        insert_patches_strategic_merge "$DIR/kustomize/kotsadm/kustomization.yaml" kotsadm-proxy.yaml
    fi

    if [ "$AIRGAP" == "1" ]; then
        if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
            cp "$DIR/addons/kotsadm/__KOTSADM_DIR__/statefulset/kotsadm-airgap.yaml" "$DIR/kustomize/kotsadm/kotsadm-airgap.yaml"
        else
            cp "$DIR/addons/kotsadm/__KOTSADM_DIR__/deployment/kotsadm-airgap.yaml" "$DIR/kustomize/kotsadm/kotsadm-airgap.yaml"
        fi
        insert_patches_strategic_merge "$DIR/kustomize/kotsadm/kustomization.yaml" kotsadm-airgap.yaml
    fi

    if [ -n "$INSTALLATION_ID" ]; then
        if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
            render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/statefulset/tmpl-kotsadm-installation-id.yaml" > "$DIR/kustomize/kotsadm/kotsadm-installation-id.yaml"
        else
            render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/deployment/tmpl-kotsadm-installation-id.yaml" > "$DIR/kustomize/kotsadm/kotsadm-installation-id.yaml"
        fi
        insert_patches_strategic_merge "$DIR/kustomize/kotsadm/kustomization.yaml" kotsadm-installation-id.yaml
    fi

    kotsadm_cacerts_file
    kotsadm_kubelet_client_secret
    kotsadm_metadata_configmap $src $dst
    kotsadm_confg_configmap $dst

    if [ -z "$KOTSADM_HOSTNAME" ]; then
        KOTSADM_HOSTNAME="$PUBLIC_ADDRESS"
    fi
    if [ -z "$KOTSADM_HOSTNAME" ]; then
        KOTSADM_HOSTNAME="$PRIVATE_ADDRESS"
    fi

    cat "$src/tmpl-start-kotsadm-web.sh" | sed "s/###_HOSTNAME_###/$KOTSADM_HOSTNAME:8800/g" > "$dst/start-kotsadm-web.sh"
    kubectl create configmap kotsadm-web-scripts --from-file="$dst/start-kotsadm-web.sh" --dry-run=client -oyaml > "$dst/kotsadm-web-scripts.yaml"

    kubectl delete pod kotsadm-migrations &> /dev/null || true;
    kubectl delete deployment kotsadm-web &> /dev/null || true; # replaced by 'kotsadm' deployment in 1.12.0
    kubectl delete service kotsadm-api &> /dev/null || true; # replaced by 'kotsadm-api-node' service in 1.12.0

    if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
        kubectl delete deployment kotsadm &> /dev/null || true; # replaced by 'kotsadm' statefulset in 1.47.1
    fi

    # removed in 1.19.0
    kubectl delete deployment kotsadm-api &> /dev/null || true
    kubectl delete service kotsadm-api-node &> /dev/null || true
    kubectl delete serviceaccount kotsadm-api &> /dev/null || true
    kubectl delete clusterrolebinding kotsadm-api-rolebinding &> /dev/null || true
    kubectl delete clusterrole kotsadm-api-role &> /dev/null || true

    # kotsadm-operator removed in 1.50.0
    kubectl delete deployment kotsadm-operator &> /dev/null || true
    kubectl delete clusterrolebinding kotsadm-operator-clusterrolebinding &> /dev/null || true
    kubectl delete clusterrole kotsadm-operator-clusterrole &> /dev/null || true
    kubectl delete serviceaccount kotsadm-operator &> /dev/null || true

    kotsadm_namespaces "$src" "$dst"

    kubectl apply -k "$dst/"

    kotsadm_kurl_proxy "$src" "$dst"

    kotsadm_postgres_ready_spinner
    kotsadm_ready_spinner

    kubectl label pvc kotsadm-postgres-kotsadm-postgres-0 velero.io/exclude-from-backup- kots.io/backup=velero --overwrite

    kotsadm_cli $src

    # Migrate existing hostpath and nfs snapshot minio to velero lvp plugin
    if [ "$KOTSADM_DISABLE_S3" == "1" ] && [ -n "$VELERO_VERSION" ] ; then
        kubectl kots velero migrate-minio-filesystems -n default
    fi
}

function kotsadm_already_applied() {

    # This prints in the outro regardless of being already applied
    if [ -z "$KOTSADM_HOSTNAME" ]; then
        KOTSADM_HOSTNAME="$PUBLIC_ADDRESS"
    fi
    if [ -z "$KOTSADM_HOSTNAME" ]; then
        KOTSADM_HOSTNAME="$PRIVATE_ADDRESS"
    fi
}

# TODO (dans): remove this when the KOTS default state is set disableS3=true
# Having no object storage in your spec and not setting disableS3 to true is invalid and not supported
function validate_object_storage() {
    if ! object_store_exists && [ "$KOTSADM_DISABLE_S3" != 1 ]; then 
        bail "KOTS must have an object storage provider as part of the installer spec (e.g. Rook or Minio), or must have 'kotsadm.disableS3=true' set in the installer"
    fi
}

function kotsadm_join() {
    kotsadm_cli "$DIR/addons/kotsadm/__KOTSADM_DIR__"
}

function kotsadm_outro() {
    local mainPod=$(kubectl get pods --selector app=kotsadm --no-headers | grep -E '(ContainerCreating|Running)' | head -1 | awk '{ print $1 }')
    if [ -z "$mainPod" ]; then
        mainPod="<main-pod>"
    fi

    printf "\n"
    printf "\n"
    printf "Kotsadm: ${GREEN}http://$KOTSADM_HOSTNAME:${KOTSADM_UI_BIND_PORT:-8800}${NC}\n"

    if [ -n "$KOTSADM_PASSWORD" ]; then
        printf "Login with password (will not be shown again): ${GREEN}$KOTSADM_PASSWORD${NC}\n"
        printf "This password has been set for you by default. It is recommended that you change this password; this can be done with the following command: ${GREEN}kubectl kots reset-password default${NC}\n"
    else
        printf "You can log in with your existing password. If you need to reset it, run ${GREEN}kubectl kots reset-password default${NC}\n"
    fi
    printf "\n"
    printf "\n"
}

function kotsadm_secret_cluster_token() {
    local CLUSTER_TOKEN=$(kubernetes_secret_value default kotsadm-cluster-token kotsadm-cluster-token)

    if [ -z "$CLUSTER_TOKEN" ]; then
        # check under old name
        CLUSTER_TOKEN=$(kubernetes_secret_value default kotsadm-auto-create-cluster-token token)

        if [ -n "$CLUSTER_TOKEN" ]; then
            kubectl delete secret kotsadm-auto-create-cluster-token
        else
            CLUSTER_TOKEN=$(< /dev/urandom tr -dc A-Za-z0-9 | head -c16)
        fi
    fi

    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-cluster-token.yaml" > "$DIR/kustomize/kotsadm/secret-cluster-token.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-cluster-token.yaml

    # ensure all pods that consume the secret will be restarted
    kotsadm_scale_down
    kubernetes_scale_down default deployment kotsadm-operator
}

function kotsadm_secret_authstring() {
    local AUTHSTRING=$(kubernetes_secret_value default kotsadm-authstring kotsadm-authstring)

    if [ -z "$AUTHSTRING" ]; then
        AUTHSTRING="Kots $(< /dev/urandom tr -dc A-Za-z0-9 | head -c32)"
    fi

    if [[ ! "$AUTHSTRING" =~ ^'Kots ' && ! "$AUTHSTRING" =~ ^'Bearer ' ]]; then
        AUTHSTRING="Kots $(< /dev/urandom tr -dc A-Za-z0-9 | head -c32)"
    fi

    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-authstring.yaml" > "$DIR/kustomize/kotsadm/secret-authstring.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-authstring.yaml
}

function kotsadm_secret_password() {
    local BCRYPT_PASSWORD=$(kubernetes_secret_value default kotsadm-password passwordBcrypt)

    if [ -z "$BCRYPT_PASSWORD" ]; then
        # global, used in outro
        KOTSADM_PASSWORD=$(< /dev/urandom tr -dc A-Za-z0-9 | head -c9)
        BCRYPT_PASSWORD=$(echo "$KOTSADM_PASSWORD" | $DIR/bin/bcrypt --cost=14)
    fi

    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-password.yaml" > "$DIR/kustomize/kotsadm/secret-password.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-password.yaml

    kotsadm_scale_down
}

function kotsadm_secret_postgres() {
    local POSTGRES_PASSWORD=$(kubernetes_secret_value default kotsadm-postgres password)

    if [ -z "$POSTGRES_PASSWORD" ]; then
        POSTGRES_PASSWORD=$(< /dev/urandom tr -dc A-Za-z0-9 | head -c16)
    fi

    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-postgres.yaml" > "$DIR/kustomize/kotsadm/secret-postgres.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-postgres.yaml

    kotsadm_scale_down
    kubernetes_scale_down default deployment kotsadm-postgres
    kubernetes_scale_down default deployment kotsadm-migrations
}

function kotsadm_secret_dex_postgres() {
    local DEX_PGPASSWORD=$(kubernetes_secret_value default kotsadm-dex-postgres PGPASSWORD)

    if [ -z "$DEX_PGPASSWORD" ]; then
        DEX_PGPASSWORD=$(< /dev/urandom tr -dc A-Za-z0-9 | head -c32)
    fi

    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-dex-postgres.yaml" > "$DIR/kustomize/kotsadm/secret-dex-postgres.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-dex-postgres.yaml

    kotsadm_scale_down
}

function kotsadm_secret_s3() {
    # When no object store is defined and S3 is disabled for KOTS, bail from adding the secret. 
    if [ -z "$OBJECT_STORE_ACCESS_KEY" ] && [ "$KOTSADM_DISABLE_S3" == "1" ]; then
        return
    fi

    if [ -z "$VELERO_LOCAL_BUCKET" ]; then
        VELERO_LOCAL_BUCKET=velero
    fi
    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-s3.yaml" > "$DIR/kustomize/kotsadm/secret-s3.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-s3.yaml
}

function kotsadm_secret_session() {
    local JWT_SECRET=$(kubernetes_secret_value default kotsadm-session key)

    if [ -z "$JWT_SECRET" ]; then
        JWT_SECRET=$(< /dev/urandom tr -dc A-Za-z0-9 | head -c16)
    fi

    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-session.yaml" > "$DIR/kustomize/kotsadm/secret-session.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-session.yaml

    kotsadm_scale_down
}

function kotsadm_api_encryption_key() {
    local API_ENCRYPTION=$(kubernetes_secret_value default kotsadm-encryption encryptionKey)

    if [ -z "$API_ENCRYPTION" ]; then
        # 24 byte key + 12 byte nonce, base64 encoded. This is separate from the base64 encoding used
        # in secrets with kubectl. Kotsadm expects the value to be encoded when read as an env var.
        API_ENCRYPTION=$(< /dev/urandom cat | head -c36 | base64)
    fi

    render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/tmpl-secret-api-encryption.yaml" > "$DIR/kustomize/kotsadm/secret-api-encryption.yaml"
    insert_resources "$DIR/kustomize/kotsadm/kustomization.yaml" secret-api-encryption.yaml

    kotsadm_scale_down
}

function kotsadm_api_patch_s3_migration() {
    insert_patches_json_6902 "$DIR/kustomize/kotsadm/kustomization.yaml" s3-migration.yaml apps v1 StatefulSet kotsadm default
    cp "$DIR/addons/kotsadm/__KOTSADM_DIR__/statefulset/patches/s3-migration.yaml" "$DIR/kustomize/kotsadm/s3-migration.yaml"
}

function kotsadm_api_patch_prometheus() {
    if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
        cp "$DIR/addons/kotsadm/__KOTSADM_DIR__/statefulset/patches/api-prometheus.yaml" "$DIR/kustomize/kotsadm/api-prometheus.yaml"
    else
        cp "$DIR/addons/kotsadm/__KOTSADM_DIR__/deployment/patches/api-prometheus.yaml" "$DIR/kustomize/kotsadm/api-prometheus.yaml"
    fi
    insert_patches_strategic_merge "$DIR/kustomize/kotsadm/kustomization.yaml" api-prometheus.yaml
}

function kotsadm_metadata_configmap() {
    local src="$1"
    local dst="$2"

    # The application.yaml pre-exists from airgap bundle OR
    # gets created below if user specified the app-slug and metadata exists.
    if [ "$AIRGAP" != "1" ] && [ -n "$KOTSADM_APPLICATION_SLUG" ]; then
        # If slug exists, but there's no branding, then replicated.app will return nothing.
        # (application.yaml will remain empty)
        echo "Retrieving app metadata: url=$REPLICATED_APP_URL, slug=$KOTSADM_APPLICATION_SLUG"
        curl $REPLICATED_APP_URL/metadata/$KOTSADM_APPLICATION_SLUG > "$src/application.yaml"
    fi
    if test -s "$src/application.yaml"; then
        cp "$src/application.yaml" "$dst/"
        kubectl create configmap kotsadm-application-metadata --from-file="$dst/application.yaml" --dry-run=client -oyaml > "$dst/kotsadm-application-metadata.yaml"
        insert_resources $dst/kustomization.yaml kotsadm-application-metadata.yaml
    fi
}

function kotsadm_confg_configmap() {
    local dst="$1"

    if ! kubernetes_resource_exists default configmap kotsadm-confg; then
        kubectl -n default create configmap kotsadm-confg
        kubectl -n default label configmap kotsadm-confg --overwrite kots.io/kotsadm=true kots.io/backup=velero
    fi

    kubectl -n default get configmap kotsadm-confg -oyaml > "$dst/kotsadm-confg.yaml"

    if [ -n "$KOTSADM_APPLICATION_VERSION_LABEL" ]; then
        "${DIR}"/bin/yamlutil -a -fp "$dst/kotsadm-confg.yaml" -yp data_app-version-label -v "$KOTSADM_APPLICATION_VERSION_LABEL"
    else
        "${DIR}"/bin/yamlutil -r -fp "$dst/kotsadm-confg.yaml" -yp data_app-version-label
    fi

    insert_resources "$dst/kustomization.yaml" kotsadm-confg.yaml
}

function kotsadm_kurl_proxy() {
    local src="$1/kurl-proxy"
    local dst="$2/kurl-proxy"

    mkdir -p "$dst"

    cp "$src/kustomization.yaml" "$dst/"
    cp "$src/rbac.yaml" "$dst/"

    render_yaml_file "$src/tmpl-service.yaml" > "$dst/service.yaml"
    render_yaml_file "$src/tmpl-deployment.yaml" > "$dst/deployment.yaml"

    kotsadm_tls_secret

    kubectl apply -k "$dst/"
}

function kotsadm_tls_secret() {
    if kubernetes_resource_exists default secret kotsadm-tls; then
        kubectl -n default label secret kotsadm-tls --overwrite kots.io/kotsadm=true kots.io/backup=velero
        return 0
    fi

    cat > kotsadm.cnf <<EOF
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
CN = kotsadm.default.svc.cluster.local

[ req_ext ]
subjectAltName = @alt_names

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:TRUE,pathlen:0
keyUsage=nonRepudiation,digitalSignature,keyEncipherment,keyCertSign
extendedKeyUsage=serverAuth
subjectAltName=@alt_names

[ alt_names ]
DNS.1 = kotsadm
DNS.2 = kotsadm.default
DNS.3 = kotsadm.default.svc
DNS.4 = kotsadm.default.svc.cluster
DNS.5 = kotsadm.default.svc.cluster.local
IP.1 = $PRIVATE_ADDRESS
EOF
    if [ -n "$PUBLIC_ADDRESS" ]; then
        echo "IP.2 = $PUBLIC_ADDRESS" >> kotsadm.cnf
    fi

    openssl req -newkey rsa:2048 -nodes -keyout kotsadm.key -config kotsadm.cnf -x509 -days 365 -out kotsadm.crt -extensions v3_ext

    kubectl -n default create secret tls kotsadm-tls --key=kotsadm.key --cert=kotsadm.crt
    kubectl -n default annotate secret kotsadm-tls acceptAnonymousUploads=1
    kubectl -n default label secret kotsadm-tls --overwrite kots.io/kotsadm=true kots.io/backup=velero

    rm kotsadm.cnf kotsadm.key kotsadm.crt
}

function kotsadm_kubelet_client_secret() {
    if kubernetes_resource_exists default secret kubelet-client-cert; then
        return 0
    fi

    kubectl -n default create secret generic kubelet-client-cert \
        --from-file=client.crt="$(${K8S_DISTRO}_get_client_kube_apiserver_crt)" \
        --from-file=client.key="$(${K8S_DISTRO}_get_client_kube_apiserver_key)" \
        --from-file="$(${K8S_DISTRO}_get_server_ca)"
}

function kotsadm_cli() {
    local src="$1"

    if ! kubernetes_is_master; then
        return 0
    fi
    if [ ! -f "$src/assets/kots.tar.gz" ] && [ "$AIRGAP" != "1" ]; then
        mkdir -p "$src/assets"
        curl -L "https://github.com/replicatedhq/kots/releases/download/__KOTSADM_BINARY_VERSION__/kots_linux_amd64.tar.gz" > "$src/assets/kots.tar.gz"
    fi

    pushd "$src/assets"
    tar xf "kots.tar.gz"
    mkdir -p "$KUBECTL_PLUGINS_PATH"
    mv kots "$KUBECTL_PLUGINS_PATH/kubectl-kots"
    popd

    # https://github.com/replicatedhq/kots/issues/149
    if [ ! -e /usr/lib64/libdevmapper.so.1.02.1 ] && [ -e /usr/lib64/libdevmapper.so.1.02 ]; then
        ln -s /usr/lib64/libdevmapper.so.1.02 /usr/lib64/libdevmapper.so.1.02.1
    fi
}

# copy pgdata from pvc named kotsadm-postgres to new pvc named kotsadm-postgres-kotsadm-postgres-0
# used by StatefulSet in 1.12.2+
function kotsadm_rename_postgres_pvc_1-12-2() {
    local src="$1"

    if kubernetes_resource_exists default deployment kotsadm-postgres; then
        kubectl delete deployment kotsadm-postgres
    fi
    if ! kubernetes_resource_exists default pvc kotsadm-postgres; then
        return 0
    fi
    printf "${YELLOW}Renaming PVC kotsadm-postgres to kotsadm-postgres-kotsadm-postgres-0${NC}\n"
    kubectl apply -f "$src/kotsadm-postgres-rename-pvc.yaml"
    spinner_until -1 kotsadm_postgres_pvc_renamed
    kubectl delete pod kotsadm-postgres-rename-pvc
    kubectl delete pvc kotsadm-postgres
}

function kotsadm_postgres_pvc_renamed {
    local status=$(kubectl get pod kotsadm-postgres-rename-pvc -ojsonpath='{ .status.containerStatuses[0].state.terminated.reason }')
    [ "$status" = "Completed" ]
}

function kotsadm_namespaces() {
    local src="$1"
    local dst="$2"

    IFS=',' read -ra KOTSADM_APPLICATION_NAMESPACES_ARRAY <<< "$KOTSADM_APPLICATION_NAMESPACES"
    for NAMESPACE in "${KOTSADM_APPLICATION_NAMESPACES_ARRAY[@]}"; do
        kubectl create ns "$NAMESPACE" 2>/dev/null || true
    done
}

function kotsadm_scale_down() {
    if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
        kubernetes_scale_down default statefulset kotsadm
    else
        kubernetes_scale_down default deployment kotsadm
    fi
}

function kotsadm_health_check() {
    local selector=$1
    # Get pods below will initially return only 0 lines
    # Then it will return 1 line: "PodScheduled=True"
    # Finally, it will return 4 lines.  And this is when we want to grep until "Ready=False" is not shown, and '1/1 Running' is
    if [ $(kubectl get pods -l ${selector} -o jsonpath="{range .items[*]}{range .status.conditions[*]}{ .type }={ .status }{'\n'}{end}{end}" 2>/dev/null | wc -l) -ne 4 ]; then
        # if this returns more than 4 lines, there are multiple copies of the pod running, which is a failure
        return 1
    fi

    if [[ -n $(kubectl get pods -l ${selector} --field-selector=status.phase=Running -o jsonpath="{range .items[*]}{range .status.conditions[*]}{ .type }={ .status }{'\n'}{end}{end}" 2>/dev/null | grep -q Ready=False) ]]; then
        # if there is a pod with Ready=False, then kotsadm is not ready
        return 1
    fi

    if [[ -z $(kubectl get pods -l ${selector} --field-selector=status.phase=Running 2>/dev/null | grep '1/1' | grep 'Running') ]]; then
        # when kotsadm is ready, it will be '1/1 Running'
        return 1
    fi
    return 0
}

function kotsadm_ready_spinner() {
    sleep 1 # ensure that kubeadm has had time to begin applying and scheduling the kotsadm pods
    if ! spinner_until 180 kotsadm_health_check "app=kotsadm"; then
      kubectl logs -l "app=kotsadm" --all-containers --tail 10
      bail "The kotsadm statefulset in the kotsadm addon failed to deploy successfully."
    fi
}

function kotsadm_postgres_ready_spinner() {
    sleep 1 # ensure that kubeadm has had time to begin applying and scheduling the kotsadm pods
    if ! spinner_until 300 kotsadm_health_check "app=kotsadm-postgres"; then
      kubectl logs -l "app=kotsadm-postgres" --all-containers --tail 10
      bail "The kotsadm-postgres statefulset in the kotsadm addon failed to deploy successfully."
    fi
}

function kotsadm_cacerts_file() {
    # Find the cacerts bundle on the host
    # if it exists, add a patch to add the volume mount to kotsadm

    # See https://github.com/golang/go/blob/ec4051763d439e7108bc673dd0b1bf1cbbc5dfc5/src/crypto/x509/root_linux.go
    # TODO(dan): need to test this re-ordering
    local sslDirectories
    sslDirectories=( \
        "/etc/ssl/certs/ca-certificates.crt" \                  # Debian/Ubuntu/Gentoo etc.
        "/etc/pki/tls/certs/ca-bundle.crt" \                    # Fedora/RHEL 6
        "/etc/ssl/ca-bundle.pem" \                              # OpenSUSE
        "/etc/pki/tls/cacert.pem" \                             # OpenELEC
        "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem" \   # CentOS/RHEL 7
        "/etc/ssl/cert.pem" \                                   # Alpine Linux
    )

    for cert_file in "${sslDirectories[@]}";  do
        if [ -f "$cert_file" ]; then
            KOTSADM_TRUSTED_CERT_MOUNT="${cert_file}"
            break
        fi
    done

    if [ -n "$KOTSADM_TRUSTED_CERT_MOUNT" ]; then
        if [ "$KOTSADM_DISABLE_S3" == "1" ]; then
            render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/statefulset/tmpl-kotsadm-cacerts.yaml" > "$DIR/kustomize/kotsadm/kotsadm-cacerts.yaml"
        else
            render_yaml_file "$DIR/addons/kotsadm/__KOTSADM_DIR__/deployment/tmpl-kotsadm-cacerts.yaml" > "$DIR/kustomize/kotsadm/kotsadm-cacerts.yaml"
        fi
        insert_patches_strategic_merge "$DIR/kustomize/kotsadm/kustomization.yaml" kotsadm-cacerts.yaml
    fi
}
