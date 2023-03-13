#!/usr/bin/env bash

##
## Script to create a development VM on GCP
##
## Options:
##  --delete             delete the VM
##  --docker             install docker
##  --k3s                install k3s
##  --k9s                install k9s
##  --kubeconfig         copy the remote kubeconfig locally
##  --kubeconfig-path    the path to copy the kubeconfig to (defaults to ~/.kube)
##  --kubeconfig-update  overwrite the local default kubeconfig with the remote one
##  --kurl               install the latest kurl
##  --name               VM name (default: $(whoami)-dev)
##  --ssh                ssh into the VM
##  --update             update all packages on the VM
##  -v | --verbose  print script commands
##

set -e

# GCP VM details, change to suit your needs
VM_REGION="us-west3"
VM_PROJECT="replicated-qa"
VM_TYPE="n1-standard-8"
VM_DISK_SIZE="500GB"
VM_IMAGE_PROJECT="ubuntu-os-cloud"
VM_IMAGE_FAMILY="ubuntu-2204-lts"

VM_USER=$(whoami)
VM_NAME="${VM_USER}-dev"

KUBECONFIG_PATH="${HOME}/.kube"

# Don't check for existing host keys
SSH_OPTS="-o StrictHostKeyChecking=no"

# Quiet the apt output
APT_OPTS="-qq -o=Dpkg::Use-Pty=0"

# Redirect output
IO_REDIRECT="/dev/null"

# Print a help message
print_help() {
    echo "Script to create, delete, and connect to a development VM on GCP"
    echo ""
    echo "Options:"
    echo "  --delete             delete the VM"
    echo "  --docker             install docker"
    echo "  --k3s                install k3s"
    echo "  --k9s                install k9s"
    echo "  --kubeconfig         copy the remote kubeconfig locally"
    echo "  --kubeconfig-path    the path to copy the kubeconfig to (default: ~/.kube)"
    echo "  --kubeconfig-update  overwrite the local default kubeconfig with the remote one"
    echo "  --kurl               install the latest kurl"
    echo "  --name               VM name (default: \$(whoami)-dev)"
    echo "  --ssh                ssh into the VM"
    echo "  --update             update all packages on the VM"
    echo "  -v | --verbose       print script commands"
}

# Print an error and exit
error() {
    print_error_message "$1"
    exit -1
}

# Print a visible error message
print_error_message() {
    msg="$1"
    printf "💀\n💀 ${msg}\n💀\n"
}

# Print a visible message
print_message() {
    printf "🔥 $1\n"
}

# Check for required utilities
command -v gcloud &> /dev/null || error "gcloud is not installed"
command -v curl &> /dev/null || error "curl is not installed"
command -v ssh &> /dev/null || error "ssh client is not installed"

# Parse options:
while [[ "$#" -gt 0 ]]; do
    case ${1} in
        --delete) DELETE_VM=1; shift ;;
        --docker) INSTALL_DOCKER=1; shift ;;
        -h | --help) print_help; exit 0 ;;
        --k3s) INSTALL_K3S=1; shift ;;
        --k9s) INSTALL_K9S=1; shift ;;
        --kubeconfig) KUBECONFIG_PULL=1; shift ;;
        --kubeconfig-path) KUBECONFIG_PATH=${2}; shift; shift ;;
        --kubeconfig-update) KUBECONFIG_UPDATE=1; shift ;;
        --kurl) INSTALL_KURL=1; shift ;;
        --name) VM_NAME=${2}; shift; shift ;;
        --ssh) SSH_VM=1; shift ;;
        --update) UPDATE_PKGS=1 ;;
        -v | --verbose)
            set -x
            IO_REDIRECT="/dev/stdout"
            unset APT_OPTS
            shift
         ;;
        *) print_error_message "unknown option: ${1}"; echo ""; print_help; exit -1 ;;
    esac
done

# If k3s is requested, docker must be installed
if [[ ${INSTALL_K3S:-0} -ne 0 ]]; then
    INSTALL_DOCKER=1
fi

# If kurl is requested, no other container or k8s runtimes can be installed
if [[ ${INSTALL_KURL:-0} -ne 0 ]]; then
    unset INSTALL_DOCKER
    unset INSTALL_K3S
fi

# If a kubeconfig update is requested, it must be pulled
if [[ ${KUBECONFIG_UPDATE:-0} -ne 0 ]]; then
    KUBECONFIG_PULL=1
fi

print_message "Searching for available zone in the ${VM_REGION} region"
VM_ZONE=$(gcloud compute --project ${VM_PROJECT} zones list | grep ${VM_REGION} | grep UP | awk 'NR==1{ print $1 }')

# Delete the instance if requested
if [[ ${DELETE_VM:-0} -ne 0 ]]; then
    print_message "Deleting ${VM_NAME} [${VM_PROJECT}] [${VM_ZONE}]"
    gcloud compute instances delete --verbosity=error --project=${VM_PROJECT} --zone=${VM_ZONE} --quiet ${VM_NAME}
    exit 0
fi

# Connect to the instance if requestd
if [[ ${SSH_VM:-0} -ne 0 ]]; then
    VM_IP=$(gcloud compute instances describe --project=${VM_PROJECT} --zone=${VM_ZONE} --format='get(networkInterfaces[0].accessConfigs[0].natIP)' ${VM_NAME})
    ssh ${VM_USER}@${VM_IP}
    exit 0
fi

# Get the latest image and available zone
print_message "Searching for latest ubuntu image"
VM_IMAGE=$(gcloud compute images list --filter="${VM_IMAGE_PROJECT} AND family~'${VM_IMAGE_FAMILY}' AND family!~'arm'" | awk 'END{ print $1 }')

# Create GCP compute instance and get its IP
print_message "Creating instance ${VM_NAME} [${VM_PROJECT}] [${VM_ZONE}] from ${VM_IMAGE}"
gcloud compute instances create ${VM_NAME} --project=${VM_PROJECT} --zone=${VM_ZONE} --machine-type=${VM_TYPE} \
    --create-disk=auto-delete=yes,boot=yes,device-name=${VM_NAME},image=${VM_IMAGE},image-project=${VM_IMAGE_PROJECT},mode=rw,size=${VM_DISK_SIZE},type=pd-ssd \
    --network-interface=network-tier=PREMIUM,subnet=default --maintenance-policy=MIGRATE --provisioning-model=STANDARD \
    --service-account=846065462912-compute@developer.gserviceaccount.com \
    --scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append \
    --quiet --verbosity=error > "${IO_REDIRECT}" || error "Creating instance ${VM_NAME} failed"

# Delete the VM after this point if there are any errors
delete_vm() {
    print_error_message "Deleting instance ${VM_NAME} [${VM_PROJECT}] [${VM_ZONE}]"
    #gcloud compute instances delete --project=${VM_PROJECT} --zone=${VM_ZONE} --quiet ${VM_NAME}
    exit -1
}
trap delete_vm ERR

# Get the IP and remove any existing known_host entries
print_message "Getting VM IP address"
VM_IP=$(gcloud compute instances describe --project=${VM_PROJECT} --zone=${VM_ZONE} --format='get(networkInterfaces[0].accessConfigs[0].natIP)' ${VM_NAME})
if [[ ! ${VM_IP+0} ]]; then
    error "No IP found for ${VM_NAME}"
fi
sed -i "" "/${VM_IP}/d" ${HOME}/.ssh/known_hosts

# Try to connect to the created VM until the ssh metadata is added
print_message "Waiting for ssh on ${VM_NAME} to be ready"
TRIES=0
while ! ssh ${SSH_OPTS} -T ${VM_USER}@${VM_IP} "exit" &> ${IO_REDIRECT} ; do
    if [[ ${TRIES} -gt 30 ]]; then
        error "Couldn't ssh into ${VM_NAME}"
    fi

    TRIES=$(( TRIES + 1 ))
    sleep 1
done

# Update packages
if [[ ${UPDATE_PKGS:-0} -ne 0 ]]; then
    print_message "Updating packages on ${VM_NAME}"
    if ! ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} "sudo apt-get ${APT_OPTS} update > ${IO_REDIRECT} && sudo apt-get ${APT_OPTS} upgrade -y > ${IO_REDIRECT}" ; then
        error "Failed to update packages on ${VM_NAME}"
    fi
fi

# Install k9s
if [[ ${INSTALL_K9S:-0} -ne 0 ]]; then
    print_message "Installing k9s on ${VM_NAME}"
    K9S_ARCHIVE=$(curl -s https://api.github.com/repos/derailed/k9s/releases/latest | jq -r '.assets[] | select(.name | test(".*Linux_amd64.*")) | .name')
    K9S_URL=$(curl -s https://api.github.com/repos/derailed/k9s/releases/latest | jq -r '.assets[] | select(.name==${K9S_ARCHIVE}) | .browser_download_url')
    ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} \
        "
        curl -LO ${K9S_URL} &&
        sudo tar xvfC ${K9S_ARCHIVE} /usr/local/bin k9s &&
        rm ${K9S_ARCHIVE}
        "
    if [[ ! $? ]]; then
        error "Failed to install k9s on ${VM_NAME}"
    fi
fi

# Install docker and containerd
if [[ ${INSTALL_DOCKER:-0} -ne 0 ]]; then
    print_message "Installing docker on ${VM_NAME}"
    # Use multiple ssh connections due to which machine the command and variable substitution happens with ' vs " quotes
    ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} "sudo mkdir -m 0755 -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg" && \
    ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} 'echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null' && \
    ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} \
        "
        sudo apt-get ${APT_OPTS} update > ${IO_REDIRECT} &&
        sudo apt-get ${APT_OPTS} install -y docker-ce docker-ce-cli containerd.io > ${IO_REDIRECT} &&
        sudo sh -c 'echo \"disabled_plugins = []\" > /etc/containerd/config.toml' &&
        sudo systemctl restart containerd
        "
    if [[ ! $? ]]; then
        error "Failed to install docker on ${VM_NAME}"
    fi
fi

pull_kubeconfig() {
    kubeconfig_full_path="${KUBECONFIG_PATH}/config.gcp.${VM_NAME}"
    print_message "Copying kubeconfig for ${VM_NAME} to ${kubeconfig_full_path}"
    if [[ ${INSTALL_K3S:-0} -ne 0 ]]; then
        if ! scp ${SSH_OPTS} ${VM_USER}@${VM_IP}:/etc/rancher/k3s/k3s.yaml ${kubeconfig_full_path}; then
            error "Failed to download kubeconfig from ${VM_NAME}"
        fi
    fi

    if [[ ${INSTALL_KURL:-0} -ne 0 ]]; then
        if ! scp ${SSH_OPTS} ${VM_USER}@${VM_IP}:/etc/kubernetes/admin.conf ${kubeconfig_full_path}; then
            error "Failed to download kubeconfig from ${VM_NAME}"
        fi
    fi

    sed -E -i "" "s/(server: https:\/\/)[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+(.*)/\1${VM_IP}\2/g" ${kubeconfig_full_path}

    if [[ ${KUBECONFIG_UPDATE:-0} -ne 0 ]]; then
        if [[ -f ${KUBECONFIG_PATH}/config ]]; then
            backup_name="config.orig-$(date +%Y.%m.%d-%H.%M.%S)"
            print_message "Overwriting kubeconfig, original backed up to ${KUBECONFIG_PATH}/${backup_name}"
            cp ${KUBECONFIG_PATH}/config ${KUBECONFIG_PATH}/${backup_name}
            cp ${kubeconfig_full_path} ${KUBECONFIG_PATH}/config
        else
            print_message "Copying kubeconfig to ${KUBECONFIG_PATH}/config"
            cp ${kubeconfig_full_path} ${KUBECONFIG_PATH}/config
        fi
    fi
}

# Configure, install, and run k3s. The 'tls-san' option causes k3s to create a tls certificate valid for it's external IP
if [[ ${INSTALL_K3S:-0} -ne 0 ]]; then
    print_message "Installing k3s on ${VM_NAME}"
    # Write the k3s config
    ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} \
        "
        sudo mkdir -p /etc/rancher/k3s &&
        echo 'write-kubeconfig-mode: 644' | sudo tee /etc/rancher/k3s/config.yaml &&
        echo 'tsl-san:' | sudo tee -a /etc/rancher/k3s/config.yaml &&
        echo '    - \"${VM_IP}\"' | sudo tee -a /etc/rancher/k3s/config.yaml
        "
    if [[ ! $? ]]; then
        error "Failed to create a k3s config on ${VM_NAME}"
    fi

    # Install k3s
    if ! ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} "curl -sfL https://get.k3s.io | sh -"; then
        error "Failed to install k3s on ${VM_NAME}"
    fi

    # On ubuntu certificates are built on demand, so demand it
    ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} "curl -vk --resolve ${VM_IP}:6443:127.0.0.1  https://${VM_IP}:6443/ping &> /dev/null" || error "Failed to generate k3s certificate"

    # Download kubeconfig
    if [[ ${KUBECONFIG_PULL:-0} -ne 0 ]]; then
        pull_kubeconfig
    fi
fi

# Install k8s and environment with kurl
if [[ ${INSTALL_KURL:-0} -ne 0 ]]; then
    print_message "Installing the k8s environment using the latest kurl release on ${VM_NAME}"
    if ! ssh ${SSH_OPTS} ${VM_USER}@${VM_IP} "curl -L https://kurl.sh/latest | sudo bash" ; then
        error "Installing the k8s environment with kurl failed on ${VM_NAME}"
    fi

    # Download and merge kubeconfig
    if [[ ${KUBECONFIG_PULL:-0} -ne 0 ]]; then
        pull_kubeconfig
    fi
fi
    
print_message "Connect to ${VM_NAME}:\n    ssh ${VM_USER}@${VM_IP}"
