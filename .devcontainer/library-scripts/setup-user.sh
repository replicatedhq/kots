#!/bin/bash
# modified from https://github.com/microsoft/vscode-dev-containers/blob/main/containers/codespaces-linux/.devcontainer/setup-user.sh
# not part of the standard script library

USERNAME=${1:-codespace}
SECURE_PATH_BASE=${2:-$PATH}

echo "Defaults secure_path=\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin:${SECURE_PATH_BASE}\"" >> /etc/sudoers.d/securepath

# Add user to a Docker group
sudo -u ${USERNAME} mkdir /home/${USERNAME}/.vsonline
groupadd -g 800 docker
usermod -a -G docker ${USERNAME}

# Create user's .local/bin
sudo -u ${USERNAME} mkdir -p /home/${USERNAME}/.local/bin
