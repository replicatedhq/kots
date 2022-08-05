import { getInput } from '@actions/core'
import { getOctokit } from '@actions/github'
import { HttpClient } from '@actions/http-client';
import fs from 'node:fs/promises';

const addonVersion = getInput('ADDON_VERSION');
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
const addonFileName = `kotsadm-${addonVersion}.tar.gz`;
kotsAddonVersions.unshift({
  kurlVersionCompatibility: latestKurlVersion.data.tag_name,
  addonUrl: `https://kots-kurl-addons-production-1658439274.s3.amazonaws.com/${addonFileName}`,
  addonVersion ,
  addonName: 'kotsadm'
});

fs.writeFile("./deploy/kurl/versions.json", JSON.stringify(kotsAddonVersions));