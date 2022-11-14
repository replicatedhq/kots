const fetch = require('node-fetch');
const fs = require('fs');
var semver = require('semver');

async function getInstallCmd() {
    // Fetch supported Kubernetes version info
    const response = await fetch("https://api.github.com/repos/helm/helm/releases/latest");
    const json = await response.json();
    const tagName = json.tag_name;
    const tagSemver = semver.parse(tagName, {loose: true});

    // Create Dockerfile command text
    const shaResponse = await fetch(`https://get.helm.sh/helm-${tagName}-linux-amd64.tar.gz.sha256sum`);
    const shasum = await shaResponse.text();
    const installCmd = `# Install Helm ${tagName}
ENV HELM3_VERSION=${tagSemver}
ENV HELM3_URL=https://get.helm.sh/helm-v${"${HELM3_VERSION}"}-linux-amd64.tar.gz
ENV HELM3_SHA256SUM=${shasum.split(" ")[0]}
RUN cd /tmp && curl -fsSL -o helm.tar.gz "${"${HELM3_URL}"}" \\
  && echo "${"${HELM3_SHA256SUM}"} helm.tar.gz" | sha256sum -c - \\
  && tar -xzvf helm.tar.gz \\
  && chmod a+x linux-amd64/helm \\
  && mv linux-amd64/helm "${"${KOTS_HELM_BIN_DIR}"}/helm${"${HELM3_VERSION}"}" \\
  && ln -s "${"${KOTS_HELM_BIN_DIR}"}/helm${"${HELM3_VERSION}"}" "${"${KOTS_HELM_BIN_DIR}"}/helm3" \\
  && ln -s "${"${KOTS_HELM_BIN_DIR}"}/helm3" "${"${KOTS_HELM_BIN_DIR}"}/helm" \\
  && rm -rf helm.tar.gz linux-amd64`;

  console.log(installCmd);
}

getInstallCmd();
