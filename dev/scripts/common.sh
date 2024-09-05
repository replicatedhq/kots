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

function up() {
  if [ "$1" == "kotsadm-web" ]; then
    # Tail the logs of the new pod
    newpod=$(kubectl get pods --no-headers --sort-by=.metadata.creationTimestamp | awk 'END {print $1}')
    kubectl logs -f $newpod --tail=100
  else
    # Exec into the deployment
    kubectl exec -it deployment/$1 -- bash
  fi
}
