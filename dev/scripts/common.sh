# Get component image name
function image() {
  jq -r ".\"$1\".image" dev/metadata.json
}

# Get component dockerfile path
function dockerfile() {
  jq -r ".\"$1\".dockerfile" dev/metadata.json
}

# Get component dockercontext
function dockercontext() {
  jq -r ".\"$1\".dockercontext" dev/metadata.json
}

# Get component deployment name
function deployment() {
  jq -r ".\"$1\".deployment" dev/metadata.json
}

# Populates local caches and binaries for faster (re)builds
function populate() {
  case "$1" in
    "kotsadm")
      docker run --rm \
        -v "$(pwd):/replicatedhq/kots" \
        -e GOCACHE=/replicatedhq/kots/dev/.gocache \
        -e GOMODCACHE=/replicatedhq/kots/dev/.gomodcache \
        -w /replicatedhq/kots \
        golang:1.23-alpine \
        /bin/sh -c "apk add make bash git && make kots build"
      ;;
    "kotsadm-web")
      docker run --rm \
        -v "$(pwd):/replicatedhq/kots" \
        -e YARN_CACHE_FOLDER=/replicatedhq/kots/dev/.yarncache \
        -w /replicatedhq/kots/web \
        node:18-alpine \
        /bin/sh -c "apk add make bash git && make deps"
      ;;
    "kurl-proxy")
      docker run --rm \
        -v "$(pwd):/replicatedhq/kots" \
        -e GOCACHE=/replicatedhq/kots/dev/.gocache \
        -e GOMODCACHE=/replicatedhq/kots/dev/.gomodcache \
        -w /replicatedhq/kots/kurl_proxy \
        golang:1.23-alpine \
        /bin/sh -c "apk add make bash git && make build"
      ;;
  esac
}

# Builds a component's image
function build() {
  docker build -t $(image $1) -f $(dockerfile $1) $(dockercontext $1)
}

# Restarts a component
function restart() {
  if [ "$1" == "kotsadm-migrations" ]; then
    kubectl delete job $1 --ignore-not-found
  elif kubectl get deployment $(deployment $1) &>/dev/null; then
    kubectl rollout restart deployment/$(deployment $1)
  fi
}

# The /host_mnt directory on Docker Desktop for macOS is a virtualized path that represents
# the mounted directories from the macOS host filesystem into the Docker Desktop VM.
# This is required for using HostPath volumes in Kubernetes.
function render() {
  PROJECT_DIR="/host_mnt$(pwd)" gomplate --missing-key zero -f "$1"
}

# The embedded-cluster container mounts the KOTS project at /replicatedhq/kots
function ec_render() {
  EC_NODE=$(ec_node) PROJECT_DIR="/replicatedhq/kots" gomplate --missing-key zero -f "$1"
}

# Get the embedded cluster node name
function ec_node() {
  echo "${EC_NODE:-node0}"
}

# Executes a command in the embedded cluster container
function ec_exec() {
  docker exec -it -w /replicatedhq/kots $(ec_node) $@
}

# Patches a component deployment in the embedded cluster
function ec_patch() {
  ec_render dev/patches/$1-up.yaml > dev/patches/$1-up-ec.yaml.tmp
  ec_exec k0s kubectl --kubeconfig=/var/lib/embedded-cluster/k0s/pki/admin.conf patch deployment $(deployment $1) -n kotsadm --patch-file dev/patches/$1-up-ec.yaml.tmp
  rm dev/patches/$1-up-ec.yaml.tmp
}

function ec_build_and_load() {
  force=$2

  # Build the image
  if [ -z "$force" ] && docker images | grep -q "$(image $1)"; then
    echo "$(image $1) image already exists, skipping build..."
  else
    echo "Building $1..."
    populate $1
    build $1
  fi

  # Load the image into the embedded cluster
  if [ -z "$force" ] && docker exec $(ec_node) k0s ctr images ls | grep -q "$(image $1)"; then
    echo "$(image $1) image already loaded in embedded cluster, skipping import..."
  else
    echo "Loading "$(image $1)" image into embedded cluster..."
    docker save "$(image $1)" | docker exec -i $(ec_node) k0s ctr images import -
  fi
}

function up() {
  # Exec into the deployment
  kubectl exec -it deployment/$(deployment $1) -- bash
}

function ec_up() {
  ec_exec k0s kubectl --kubeconfig=/var/lib/embedded-cluster/k0s/pki/admin.conf exec -it deployment/$(deployment $1) -n kotsadm -- bash
}
