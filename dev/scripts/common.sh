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
  sed "s|__PROJECT_DIR__|/host_mnt$(pwd)|g" "$1"
}

# The embedded-cluster container mounts the KOTS project at /replicatedhq/kots
function render_ec() {
  sed "s|__PROJECT_DIR__|/replicatedhq/kots|g" "$1"
}

# Executes a command in the embedded cluster container
function exec_ec() {
  docker exec -it -w /replicatedhq/kots node0 $@
}

# Patches a component deployment in the embedded cluster
function patch_ec() {
  render_ec dev/patches/$1-up.yaml > dev/patches/$1-up-ec.yaml.tmp
  exec_ec k0s kubectl patch deployment $(deployment $1) -n kotsadm --patch-file dev/patches/$1-up-ec.yaml.tmp
  rm dev/patches/$1-up-ec.yaml.tmp
}

function build_and_load_ec() {
  # Build the image
  if docker images | grep -q "$(image $1)"; then
    echo "$(image $1) image already exists, skipping build..."
  else
    echo "Building $1..."
    docker build -t $(image $1) -f $(dockerfile $1) $(dockercontext $1)
  fi

  # Load the image into the embedded cluster
  if docker exec node0 k0s ctr images ls | grep -q "$(image $1)"; then
    echo "$(image $1) image already loaded in embedded cluster, skipping import..."
  else
    echo "Loading "$(image $1)" image into embedded cluster..."
    docker save "$(image $1)" | docker exec -i node0 k0s ctr images import -
  fi
}

function up() {
  if [ "$1" == "kotsadm-web" ]; then
    # Tail the logs of the new pod
    newpod=$(kubectl get pods --no-headers --sort-by=.metadata.creationTimestamp | awk 'END {print $1}')
    kubectl logs -f $newpod --tail=100
  else
    # Exec into the deployment
    kubectl exec -it deployment/$(deployment $1) -- bash
  fi
}

function up_ec() {
  exec_ec k0s kubectl exec -it deployment/$(deployment $1) -n kotsadm -- bash
}
