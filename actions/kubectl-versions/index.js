const fetch = require('node-fetch');
const fs = require('fs');
const yaml = require('js-yaml');

async function updateVersions() {
    // Fetch supported Kubernetes version info
    const latestSupportedVersions = await fetchLatestSupportedVersions();

    // Create Dockerfile command text
    let i = 0
    for (const v of latestSupportedVersions) {
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
        if (i === latestSupportedVersions.length - 1) {
            v.install = v.install + ` \\
  && ln -s "${"${KOTS_KUBECTL_BIN_DIR}"}/kubectl-v${v.cycle}" "${"${KOTS_KUBECTL_BIN_DIR}"}/kubectl"`
        }
        i++
    }

    const commandText = latestSupportedVersions.map(s => s.install).join("\n\n");

    // Insert command text into Dockerfile
    fs.readFile("deploy/Dockerfile", 'utf8', function (err, data) {
        if (err) {
            return console.error(err);
        }
        var result = data.replace(/(# __BEGIN_KUBECTL_VERSIONS__\n\n)([\s\S]*?)(\n\n# __END_KUBECTL_VERSIONS__)/g, `$1${commandText}$3`);

        fs.writeFile("deploy/Dockerfile", result, 'utf8', function (err) {
            if (err) return console.error(err);
        });
    });
}

async function fetchLatestSupportedVersions() {
    const response = await fetch("https://raw.githubusercontent.com/kubernetes/website/main/data/releases/schedule.yaml");
    const responseYAML = await response.text();
    const responseJSON = yaml.load(responseYAML);
    const supported = responseJSON.schedules.map(schedule => {
        let latest = "";
        if (!schedule.previousPatches || schedule.previousPatches.length === 0) {
            // There are no patch releases
            latest = schedule.release + ".0"; // append the patch version of .0 since it's the first release in the cycle
        } else {
            // There are patch releases, the first with be the latest
            latest = schedule.previousPatches[0].release;
        }
        return {
            cycle: schedule.release.toString(),
            latest: latest
        };
    }).reverse(); // return in order of oldest to newest minor version

    return supported;
}

updateVersions();
