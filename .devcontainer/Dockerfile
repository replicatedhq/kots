#-------------------------------------------------------------------------------------------------------------
# Modified from Codespaces default container image: https://github.com/microsoft/vscode-dev-containers/blob/main/containers/codespaces-linux/history/1.6.3.md
# - Remove PHP, Ruby, Dotnet, Java, powershell, rust dependencies
# - Remove fish shell
# - Remove Oryx
# - Remove git-lfs
# - Change shell to zsh
#
# TODO (dans): find a better way to pull in library script dynamically from vscode repo
# TODO (dans): AWS CLI - make a common script in the dev-containers repo
# TODO (dans): Gcloud CLI - make a common script in the dev-containers repo
# TODO (dans): add gcommands alias
# TODO (dans): terraform 
#-------------------------------------------------------------------------------------------------------------
FROM mcr.microsoft.com/oryx/build:vso-focal-20210902.1 as replicated

ARG USERNAME=codespace
ARG USER_UID=1000
ARG USER_GID=$USER_UID
ARG HOMEDIR=/home/$USERNAME

ARG GO_VERSION="latest"

# Default to bash shell (other shells available at /usr/bin/fish and /usr/bin/zsh)
ENV SHELL=/bin/bash \
    ORYX_ENV_TYPE=vsonline-present \
    NODE_ROOT="${HOMEDIR}/.nodejs" \
    PYTHON_ROOT="${HOMEDIR}/.python" \
    HUGO_ROOT="${HOMEDIR}/.hugo" \
    NVM_SYMLINK_CURRENT=true \
    NVM_DIR="/home/${USERNAME}/.nvm" \
    NVS_HOME="/home/${USERNAME}/.nvs" \
    NPM_GLOBAL="/home/${USERNAME}/.npm-global" \
    KREW_HOME="/home/${USERNAME}/.krew/bin" \
    PIPX_HOME="/usr/local/py-utils" \
    PIPX_BIN_DIR="/usr/local/py-utils/bin" \
    GOROOT="/usr/local/go" \
    GOPATH="/go" 

ENV PATH="${PATH}:${KREW_HOME}:${NVM_DIR}/current/bin:${NPM_GLOBAL}/bin:${ORIGINAL_PATH}:${GOROOT}/bin:${GOPATH}/bin:${PIPX_BIN_DIR}:/opt/conda/condabin:${NODE_ROOT}/current/bin:${PYTHON_ROOT}/current/bin:${HUGO_ROOT}/current/bin:${ORYX_PATHS}"

COPY library-scripts/* first-run-notice.txt /tmp/scripts/
COPY ./config/* /etc/replicated/
COPY ./lifecycle-scripts/* /var/lib/replicated/scripts/

# Install needed utilities and setup non-root user. Use a separate RUN statement to add your own dependencies.
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    # Restore man command
    && yes | unminimize 2>&1 \ 
    # Run common script and setup user
    && bash /tmp/scripts/common-debian.sh "true" "${USERNAME}" "${USER_UID}" "${USER_GID}" "true" "true" "true" \
    && bash /tmp/scripts/setup-user.sh "${USERNAME}" "${PATH}" \
    # Change owner of opt contents since Oryx can dynamically install and will run as "codespace"
    && chown ${USERNAME} /opt/* \
    && chsh -s /bin/bash ${USERNAME} \
    # Verify expected build and debug tools are present
    && apt-get -y install build-essential cmake python3-dev \
    # Install tools and shells not in common script
    && apt-get install -yq vim vim-doc xtail software-properties-common libsecret-1-dev \
    # Install additional tools (useful for 'puppeteer' project)
    && apt-get install -y --no-install-recommends libnss3 libnspr4 libatk-bridge2.0-0 libatk1.0-0 libx11-6 libpangocairo-1.0-0 \
                                                  libx11-xcb1 libcups2 libxcomposite1 libxdamage1 libxfixes3 libpango-1.0-0 libgbm1 libgtk-3-0 \
    && bash /tmp/scripts/sshd-debian.sh \
    && bash /tmp/scripts/github-debian.sh \
    && bash /tmp/scripts/azcli-debian.sh \
    # Install Moby CLI and Engine
    && /bin/bash /tmp/scripts/docker-debian.sh "true" "/var/run/docker-host.sock" "/var/run/docker.sock" "${USERNAME}" "true" \
    # && bash /tmp/scripts/docker-in-docker-debian.sh "true" "${USERNAME}" "true" \
    && bash /tmp/scripts/kubectl-helm-debian.sh \
    # Build latest git from source
    && bash /tmp/scripts/git-from-src-debian.sh "latest" \
    # Clean up
    && apt-get autoremove -y && apt-get clean -y \
    # Move first run notice to right spot
    && mkdir -p /usr/local/etc/vscode-dev-containers/ \
    && mv -f /tmp/scripts/first-run-notice.txt /usr/local/etc/vscode-dev-containers/

# Install Python
RUN bash /tmp/scripts/python-debian.sh "none" "/opt/python/latest" "${PIPX_HOME}" "${USERNAME}" "true" \
    && apt-get clean -y

# Setup Node.js, install NVM and NVS
RUN bash /tmp/scripts/node-debian.sh "${NVM_DIR}" "none" "${USERNAME}" \
    && (cd ${NVM_DIR} && git remote get-url origin && echo $(git log -n 1 --pretty=format:%H -- .)) > ${NVM_DIR}/.git-remote-and-commit \
    # Install nvs (alternate cross-platform Node.js version-management tool)
    && sudo -u ${USERNAME} git clone -c advice.detachedHead=false --depth 1 https://github.com/jasongin/nvs ${NVS_HOME} 2>&1 \
    && (cd ${NVS_HOME} && git remote get-url origin && echo $(git log -n 1 --pretty=format:%H -- .)) > ${NVS_HOME}/.git-remote-and-commit \
    && sudo -u ${USERNAME} bash ${NVS_HOME}/nvs.sh install \
    && rm ${NVS_HOME}/cache/* \
    # Set npm global location
    && sudo -u ${USERNAME} npm config set prefix ${NPM_GLOBAL} \
    && npm config -g set prefix ${NPM_GLOBAL} \
    # Clean up
    && rm -rf ${NVM_DIR}/.git ${NVS_HOME}/.git

# Install Go
RUN bash /tmp/scripts/go-debian.sh "${GO_VERSION}" "${GOROOT}" "${GOPATH}" "${USERNAME}" 

# Install Replicated Tools
RUN bash /tmp/scripts/replicated-debian.sh \
    && rm -rf /tmp/scripts \
    && apt-get clean -y 

# Userspace
ENV SHELL=/bin/zsh
USER ${USERNAME}
COPY --chown=${USERNAME}:root library-scripts/replicated-userspace.sh /tmp/scripts/
RUN bash /usr/local/share/docker-init.sh \
    && bash /tmp/scripts/replicated-userspace.sh \
    && rm -rf /tmp/scripts/scripts

# Fire Docker/Moby script if needed along with Oryx's benv
ENTRYPOINT [ "/usr/local/share/docker-init.sh", "/usr/local/share/ssh-init.sh", "benv" ]
CMD [ "sleep", "infinity" ]

