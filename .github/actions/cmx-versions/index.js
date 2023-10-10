const core = require('@actions/core');
const fetch = require('node-fetch');
const semverCoerce = require('semver/functions/coerce');
const semverMajor = require('semver/functions/major');
const semverMinor = require('semver/functions/minor');
const semverGt = require('semver/functions/gt');

async function getClusterVersions() {
    const url = 'https://api.replicated.com/vendor/v3/cluster/versions';
    const apiToken = core.getInput('replicated-api-token') || process.env.REPLICATED_API_TOKEN;
    const headers = {
        Authorization: apiToken
    };

    let clusterVersions = [];
    try {
        const response = await fetch(url, {
            method: 'GET',
            headers,
        });

        if (response.status === 200) {
            const payload = await response.json();
            clusterVersions = payload['cluster-versions'];
        } else {
            throw new Error(`Request failed with status code ${response.status}`);
        }
    } catch (error) {
        console.error(`Error: ${error.message}`);
        core.setFailed(error.message);
        return;
    }

    // versions to test looks like this:
    // [
    //   {distribution: k3s, version: v1.24, stage: 'stable'},
    //   ...
    // ]
    const versionsToTest = [];

    clusterVersions.forEach((distribution) => {
        const distroName = distribution.short_name;

        if (distroName === 'helmvm' || distroName === 'kurl') {
            // excluding the embedded distributions
            return;
        }

        const latestMinorVersions = {};
        distribution.versions.forEach((version) => {
            const parsed = semverCoerce(version);
            const majorMinor = `${semverMajor(parsed)}.${semverMinor(parsed)}`;
            if (latestMinorVersions[distroName] === undefined) {
                latestMinorVersions[distroName] = {
                    [majorMinor]: version,
                };
            } else if (latestMinorVersions[distroName][majorMinor] === undefined) {
                latestMinorVersions[distroName][majorMinor] = version;
            } else {
                const currentVersion = latestMinorVersions[distroName][majorMinor];
                if (semverGt(parsed, semverCoerce(currentVersion))) {
                    latestMinorVersions[distroName][majorMinor] = version;
                }
            }
        });

        Object.keys(latestMinorVersions[distroName]).forEach((minorVersion) => {
            let stage = 'stable';
            if (distroName === 'openshift' && minorVersion === '4.10') {
                stage = 'beta';
            }

            versionsToTest.push({ distribution: distroName, version: latestMinorVersions[distroName][minorVersion], stage });
        });
    });

    console.log(versionsToTest);
    core.setOutput('versions-to-test', JSON.stringify(versionsToTest));
}

getClusterVersions();
