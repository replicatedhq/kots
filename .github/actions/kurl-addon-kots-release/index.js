import { getInput } from '@actions/core'
import { HttpClient } from '@actions/http-client';

const addonVersion = getInput('ADDON_VERSION');
const client = new HttpClient();

let kotsAddonVersions = await client.get('https://kots-kurl-addons-production-1658439274.s3.amazonaws.com/versions.json')
.then(response => response.readBody())
.then(response => JSON.parse(response));

let foundPrerelease = false;
kotsAddonVersions.forEach((addon) => {
    if (addon.version === addonVersion && addon.isPrerelease) {
        addon.isPrerelease = false;
        foundPrerelease = true;
    }
});

if (!foundPrerelease) {
    throw new Error(`Could not find addon version ${addonVersion} prerelease in versions.json`);
}


fs.writeFile('./deploy/kurl/versions.json', JSON.stringify(kotsAddonVersions));
