const fetch = require('node-fetch');
const fs = require('fs');
var semver = require('semver');


const OLDEST_SUPPORTED_VERSION = "1.19.0"

async function updateVersions() {
    // Fetch supported Kubernetes version info
    const response = await fetch("https://endoflife.date/api/kubernetes.json");
    const releases = await response.json();

    const supported = releases.filter(release => semver.gte(release.latest, OLDEST_SUPPORTED_VERSION)).reverse();
    // NOTE: The following could be used if we want to only include officially supported versions
    // const supported = releases.filter(release => new Date(release.eol) > new Date()).reverse();

    // Create Dockerfile command text
    let i = 0
    for (const v of supported) {
        const shaResponse = await fetch(`https://dl.k8s.io/release/v${v.latest}/bin/linux/amd64/kubectl.sha256`);
        const shasum = await shaResponse.text();
        v.install = `# Install Kubectl ${v.cycle}
ENV KUBECTL_${v.cycle.replace(".", "_")}_VERSION=v${v.latest}
ENV KUBECTL_${v.cycle.replace(".", "_")}_URL=https://dl.k8s.io/release/${"${KUBECTL_" + v.cycle.replace(".", "_") + "_VERSION}"}/bin/linux/amd64/kubectl
ENV KUBECTL_${v.cycle.replace(".", "_")}_SHA256SUM=${shasum}
RUN curl -fsSLO "${"${KUBECTL_" + v.cycle.replace(".", "_") + "_URL}"}" \\
  && echo "${"${KUBECTL_" + v.cycle.replace(".", "_") + "_SHA256SUM}"} kubectl" | sha256sum -c - \\
  && chmod +x kubectl \\
  && mv kubectl "${"${KOTS_KUBECTL_BIN_DIR}"}/kubectl-v${v.cycle}"`;
        if (i === supported.length - 1) {
            v.install = v.install + ` \\
  && ln -s "${"${KOTS_KUBECTL_BIN_DIR}"}/kubectl-v${v.cycle}" "${"${KOTS_KUBECTL_BIN_DIR}"}/kubectl"`
        }
        i++
    }

    const commandText = supported.map(s => s.install).join("\n\n");

    // Insert command text into template and overwrite Dockerfile
    fs.readFile("actions/kubectl-versions/template/Dockerfile.template", 'utf8', function (err, data) {
        if (err) {
            return console.error(err);
        }
        var result = data.replace(/__KUBECTL_VERSIONS__/g, commandText);

        fs.writeFile("deploy/Dockerfile", result, 'utf8', function (err) {
            if (err) return console.error(err);
        });
    });
}

updateVersions();
