import { execSync } from 'child_process';

export async function downloadAirgapBundle(
  customerID: string,
  channelSequence: number,
  portalBase64Password: string,
  destPath: string
) {
  // get airgap bundle download url
  const getBundleURLCommand = `curl -XGET "https://api.replicated.com/market/v3/airgap/images/url?customer_id=${customerID}&channel_sequence=${channelSequence}" -H 'Authorization: Basic ${portalBase64Password}'`;
  console.log(getBundleURLCommand, "\n");
  let output = execSync(getBundleURLCommand).toString();
  const bundleUrl = JSON.parse(output).url

  // download airgap bundle
  const downloadCommand = `curl '${bundleUrl}' -o ${destPath}`;
  execSync(downloadCommand, {stdio: 'inherit'});
}

export async function listReleases(
  appId: string,
  channelId: string
) {
  const response = await fetch(
    `https://api.replicated.com/vendor/v3/app/${appId}/channel/${channelId}/releases`,
    {
      headers: {
        'Authorization': process.env.REPLICATED_API_TOKEN!,
        'Content-Type': 'application/json'
      }
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to list releases: ${response.status}`);
  }

  return (await response.json()).releases;
}

export async function promoteRelease(
  appId: string,
  releaseSequence: number,
  channelId: string,
  versionLabel: string
) {
  const response = await fetch(
    `https://api.replicated.com/vendor/v3/app/${appId}/release/${releaseSequence}/promote`,
    {
      method: 'POST',
      headers: {
        'Authorization': process.env.REPLICATED_API_TOKEN!,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        channelIds: [
          channelId
        ],
        versionLabel: versionLabel
      })
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to promote vendor release sequence: ${response.status}`);
  }
}
