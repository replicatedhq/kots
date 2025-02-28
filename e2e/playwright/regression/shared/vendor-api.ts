import { VENDOR_APP_ID } from './constants';

export async function promoteVendorRelease(
  releaseSequence: number,
  channelId: string,
  versionLabel: string
) {
  const response = await fetch(
    `https://api.replicated.com/vendor/v3/app/${VENDOR_APP_ID}/release/${releaseSequence}/promote`,
    {
      method: 'POST',
      headers: {
        'Authorization': process.env.VENDOR_API_TOKEN!,
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

  return response;
}
