import { getInput } from '@actions/core'
import { getOctokit } from '@actions/github'
import { HttpClient } from '@actions/http-client';
import fs from 'node:fs/promises';

const addonVersion = getInput('ADDON_VERSION');
const addonPackageUrl = getInput('ADDON_PACKAGE_URL');
const githubToken = getInput('GITHUB_TOKEN');
const github = getOctokit(githubToken);
const client = new HttpClient();

const latestKurlVersion = await github.rest.repos.getLatestRelease({
  owner: 'replicatedhq',
  repo: 'kurl'
});
const kotsAddonVersions = await client.get('https://kots-kurl-addons-production-1658439274.s3.amazonaws.com/versions.json')
  .then(response => response.readBody())
  .then(response => JSON.parse(response));

const appendVersion = (kotsAddonVersions, version) => {
  kotsAddonVersions = kotsAddonVersions.filter(el => el.version !== version.version);
  kotsAddonVersions.unshift(version);
  return kotsAddonVersions.sort((a, b) => {
    if (a.version < b.version) {
      return -1;
    } else if (a.version > b.version) {
      return 1;
    }
    return 0;
  }).reverse();
};

kotsAddonVersions = appendVersion(kotsAddonVersions, {
  version: addonVersion,
  url: addonPackageUrl,
  // Be careful when regenerating the kURL add-on package as the kurlVersionCompatibilityRange
  // will be updated to the latest kURL version.
  kurlVersionCompatibilityRange: `>= ${latestKurlVersion.data.tag_name}`,
});

fs.writeFile('./deploy/kurl/versions.json', JSON.stringify(kotsAddonVersions));
