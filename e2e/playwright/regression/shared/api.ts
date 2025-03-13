import { VENDOR_APP_ID } from './constants';
import { runCommandWithOutput, downloadViaJumpbox } from './cli';

export async function promoteRelease(
  releaseSequence: number,
  channelId: string,
  versionLabel: string,
  releaseNotes?: string
): Promise<void> {
  const response = await fetch(
    `https://api.replicated.com/vendor/v3/app/${VENDOR_APP_ID}/release/${releaseSequence}/promote`,
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
        versionLabel: versionLabel,
        releaseNotes: releaseNotes
      })
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to promote vendor release sequence: ${response.status}`);
  }
}

export async function updateCustomer(
  customerId: string,
  customerName: string,
  channelId: string,
  isAirgapSupported: boolean,
  isEC: boolean,
  intEntitlement: number
): Promise<void> {
  const response = await fetch(
    `https://api.replicated.com/vendor/v3/customer/${customerId}`,
    {
      method: 'PUT',
      headers: {
        'Authorization': process.env.REPLICATED_API_TOKEN!,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        app_id: VENDOR_APP_ID,
        avatar: "",
        channel_id: channelId,
        domain: "",
        email: "qakotsregression@replicated.com",
        entitlementValues: [{
          name: "int_field_1",
          value: intEntitlement.toString()
        }],
        expires_at: "",
        is_airgap_enabled: isAirgapSupported,
        is_geoaxis_supported: false,
        is_gitops_supported: true,
        is_identity_service_supported: true,
        is_snapshot_supported: true,
        is_kots_install_enabled: true,
        is_kurl_install_enabled: true,
        is_embedded_cluster_download_enabled: isEC,
        is_disaster_recovery_supported: isEC,
        name: customerName,
        type: "prod"
      })
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to update license: ${response.status}`);
  }
}

export async function downloadAirgapBundle(
  customerID: string,
  channelSequence: number,
  portalBase64Password: string,
  destPath: string
) {
  // get airgap bundle download url
  const output = runCommandWithOutput(`curl -XGET 'https://api.replicated.com/market/v3/airgap/images/url?customer_id=${customerID}&channel_sequence=${channelSequence}' -H 'Authorization: Basic ${portalBase64Password}'`, true);
  const bundleUrl = JSON.parse(output).url;

  // download airgap bundle through jumpbox
  downloadViaJumpbox(bundleUrl, destPath);
}

export async function updateIdentityServiceOktaApp(oktaDomain: string, oktaAppId: string): Promise<any> {
  let response = await fetch(
    `https://${oktaDomain}/api/v1/apps/${oktaAppId}`,
    {
      method: 'GET',
      headers: {
        'Authorization': `SSWS ${process.env.IDENTITY_SERVICE_OKTA_ACCESS_TOKEN!}`
      }
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to get okta app: ${response.status}`);
  }

  const app = await response.json();
  app.settings.oauthClient.redirect_uris = [`${process.env.BASE_URL}/dex/callback`];
  app.settings.oauthClient.post_logout_redirect_uris = [process.env.BASE_URL];

  response = await fetch(
    `https://${oktaDomain}/api/v1/apps/${oktaAppId}`,
    {
      method: 'PUT',
      headers: {
        'Authorization': `SSWS ${process.env.IDENTITY_SERVICE_OKTA_ACCESS_TOKEN!}`,
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(app)
    }
  );

  if (!response.ok) {
    throw new Error(`Failed to update okta app: ${response.status}`);
  }
}
