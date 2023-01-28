import { getInput } from '@actions/core'
import { getOctokit } from '@actions/github'
import { HttpClient } from '@actions/http-client';
import * as semverCompare from 'semver/functions/compare';
import fs from 'node:fs/promises';
import { appendVersion } from './publisher';

const addonVersion = getInput('ADDON_VERSION');
const addonPackageUrl = getInput('ADDON_PACKAGE_URL');
const githubToken = getInput('GITHUB_TOKEN');
const github = getOctokit(githubToken);
const client = new HttpClient();

const latestKurlVersion = await github.rest.repos.getLatestRelease({
  owner: 'replicatedhq',
  repo: 'kurl'
});
let kotsAddonVersions = await client.get('https://kots-kurl-addons-production-1658439274.s3.amazonaws.com/versions.json')
  .then(response => response.readBody())
  .then(response => JSON.parse(response));

kotsAddonVersions = appendVersion(kotsAddonVersions, {
  version: addonVersion,
  url: addonPackageUrl,
  isPrerelease: true, // this will be updated to false when the addon is no longer a prerelease
  // Be careful when regenerating the kURL add-on package as the kurlVersionCompatibilityRange
  // will be updated to the latest kURL version.
  kurlVersionCompatibilityRange: `>= ${latestKurlVersion.data.tag_name}`,
});

fs.writeFile('./deploy/kurl/versions.json', JSON.stringify(kotsAddonVersions));
