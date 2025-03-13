const core = require('@actions/core');
const fetch = require('node-fetch');
const semverCoerce = require('semver/functions/coerce');
const semverMajor = require('semver/functions/major');
const semverMinor = require('semver/functions/minor');
const semverGt = require('semver/functions/gt');
const semverRSort = require('semver/functions/rsort');

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

    let filters = {
        k3s: {
            latest_minor_versions: true,
            numOfLatestVersions: 3
        },
        eks: {
            // latest_version: true,
            // TODO: re-enable latest_version once we have compatibility with 1.30
            versions: new Set(["1.29"]),
            instance_type: "m7g.large" // arm64
        },
        openshift: {
            // filtering out all versions except 4.15.0-okd for now per sc-90893
            versions: new Set(["4.15.0-okd"])
        },
        // disable oke for now per sc-120817
        // oke: {
        //     versions: new Set(["1.30.1"])
        // }
    }

    // versions to test looks like this:
    // [
    //   {distribution: k3s, version: v1.24, stage: 'stable', instance_type: '']},
    //   {distribution: eks, version: v1.28, stage: 'stable', instance_type: 'm7g.large']},
    //   ...
    // ]
    const versionsToTest = [];

    clusterVersions.forEach((distribution) => {
        const distroName = distribution.short_name;

        if (filters[distroName] === undefined) {
            // no filters for this distribution, skip it
            return;
        }

        let stage = 'stable';
        if (distroName === 'aks') {
            stage = 'alpha';
        }

        let instanceType = '';
        if (filters[distroName].instance_type !== undefined) {
            instanceType = filters[distroName].instance_type;
        }

        if (filters[distroName].versions !== undefined) {
            // specific versions
            const filterVersions = filters[distroName].versions;
            distribution.versions.forEach((version) => {
                if (filterVersions.has(version)) {
                    versionsToTest.push({ distribution: distroName, version, instance_type: instanceType, stage });
                }
            });
        }

        if (!!filters[distroName].latest_version) {
            // latest version
            const latestVersion = getLatestVersion(distribution);
            versionsToTest.push({ distribution: distroName, version: latestVersion, instance_type: instanceType, stage });
        }

        if (!!filters[distroName].latest_minor_versions) {
            // latest minor versions
            const latestMinorVersions = getLatestMinorVersions(distribution, getLastMajorMinorVersions(distribution, filters[distroName].numOfLatestVersions)  );
            Object.keys(latestMinorVersions).forEach((minorVersion) => {
                versionsToTest.push({ distribution: distroName, version: latestMinorVersions[minorVersion], instance_type: instanceType, stage });
            });
        }
    });

    console.log(versionsToTest);
    core.setOutput('versions-to-test', JSON.stringify(versionsToTest));
}

function getLatestVersion(distribution) {
    let latestVersion = undefined;
    distribution.versions.forEach((version) => {
        if (latestVersion === undefined) {
            latestVersion = version;
        } else {
            const parsed = semverCoerce(version);
            if (semverGt(parsed, semverCoerce(latestVersion))) {
                latestVersion = version;
            }
        }
    });

    return latestVersion;
}

function getLastMajorMinorVersions(distribution, numOfLatestMajorMinorVersions) {
    var latestMajorMinorVersions = [];
    distribution.versions.forEach((version) => {
        const parsed = semverCoerce(version);
        latestMajorMinorVersions.push(`${semverMajor(parsed)}.${semverMinor(parsed)}`);
    });
    latestMajorMinorVersions = sortVersions(latestMajorMinorVersions);
    return latestMajorMinorVersions.slice(0, numOfLatestMajorMinorVersions);
}

function sortVersions(latestMajorMinorVersions) {
    latestMajorMinorVersions = [...new Set(latestMajorMinorVersions)];
    latestMajorMinorVersions = latestMajorMinorVersions.map(version => version + ".0");
    latestMajorMinorVersions = semverRSort(latestMajorMinorVersions);
    latestMajorMinorVersions = latestMajorMinorVersions.map(version => version.replace('.0', ''));
    return latestMajorMinorVersions;
}

function getLatestMinorVersions(distribution, majorMinorVersionFilter) {
    const latestMinorVersions = {};
    distribution.versions.forEach((version) => {
        const parsed = semverCoerce(version);
        // Check if majorVersions is null, undefined, empty, or includes the current version's major
        const majorMinor = `${semverMajor(parsed)}.${semverMinor(parsed)}`;
        if(!majorMinorVersionFilter?.length || majorMinorVersionFilter.includes(majorMinor)) {
            if (latestMinorVersions[majorMinor] === undefined) {
                latestMinorVersions[majorMinor] = version;
            } else {
                const currentVersion = latestMinorVersions[majorMinor];
                if (semverGt(parsed, semverCoerce(currentVersion))) {
                    latestMinorVersions[majorMinor] = version;
                }
            }
        }


    });
    return latestMinorVersions;
}

getClusterVersions();
