import { test, expect, Page, FrameLocator } from '@playwright/test';
import { login } from '../shared';

const appUpgradeVersion = process.env.APP_UPGRADE_VERSION;

test.skip(!appUpgradeVersion, 'APP_UPGRADE_VERSION is not set');

// This test reproduces the embedded-cluster deploy-upgrade flow that exercises the
// KOTS upgrade-service iframe. It is a regression guard against headers or other
// KOTS changes that would prevent the iframe from rendering the config UI.
//
// Required environment variables:
//   APP_UPGRADE_VERSION    - label of the available update shown in the UI
//   APP_INITIAL_HOSTNAME   - (optional) expected initial hostname value
//   SKIP_CLUSTER_UPGRADE_CHECK - (optional) set to skip the cluster-update modal check

test('upgrade-service iframe renders the config form', async ({ page }) => {
  test.setTimeout(15 * 60 * 1000); // 15 minutes
  await page.goto('/');
  await login(page);
  await runDeployUpgradeWithRetry(page);
  await verifyUpgradeSuccess(page);
});

async function runDeployUpgradeWithRetry(page: Page, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    await initiateUpgrade(page);
    const iframe = page.frameLocator('#upgrade-service-iframe');
    await fillConfigForm(iframe);
    await handlePreflightChecks(iframe);
    await deployUpgrade(iframe);
    await waitForClusterUpdate(page);

    // Check for the transient "Upgrade failed" modal (e.g. 404 on binary download)
    const failedModal = page.locator('dialog, .Modal-body').filter({ hasText: 'Upgrade failed' });
    try {
      await failedModal.waitFor({ timeout: 5_000 });
    } catch (e) {
      // Only a timeout means the modal did not appear — upgrade succeeded
      if (e instanceof Error && e.name === 'TimeoutError') {
        return;
      }
      throw e;
    }

    // Modal was found — dismiss it and retry the full deploy flow
    await page.getByRole('button', { name: 'Ok, got it!' }).click();
    await expect(failedModal).not.toBeVisible({ timeout: 5_000 });
  }
  throw new Error(`Deploy upgrade failed after ${maxRetries} retries due to transient errors`);
}

async function initiateUpgrade(page: Page) {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();
  await page.locator('.available-update-row', { hasText: appUpgradeVersion }).getByRole('button', { name: 'Deploy', exact: true }).click();
}

async function fillConfigForm(iframe: FrameLocator) {
  // More precise than the original broad `h3` locator; this fails with a clear
  // message if the iframe is blocked by X-Frame-Options or the config UI fails to load.
  await expect(iframe.getByRole('heading', { name: 'The First Config Group' })).toBeVisible({ timeout: 60 * 1000 });

  const hostnameInput = iframe.locator('#hostname-group').locator('input[type="text"]');
  await expect(hostnameInput).toHaveValue(
    process.env.APP_INITIAL_HOSTNAME
      ? process.env.APP_INITIAL_HOSTNAME
      : /(initial|updated)-hostname\.com/
  );
  await hostnameInput.click();
  await hostnameInput.fill('updated-hostname.com');

  await iframe.locator('input[type="password"]').click();
  await iframe.locator('input[type="password"]').fill('updated password');

  await iframe.getByRole('button', { name: 'Next', exact: true }).click();
}

async function handlePreflightChecks(iframe: FrameLocator) {
  await expect(iframe.getByText('Preflight checks', { exact: true })).toBeVisible({ timeout: 30 * 1000 });
  await expect(iframe.getByRole('button', { name: 'Rerun' })).toBeVisible({ timeout: 30 * 1000 });
  await expect(iframe.locator('#app')).toContainText('The Volume Snapshots CRD exists');
  await expect(iframe.getByRole('button', { name: 'Back: Config' })).toBeVisible();
  await iframe.getByRole('button', { name: 'Next: Confirm and deploy' }).click();
}

async function deployUpgrade(iframe: FrameLocator) {
  await expect(iframe.locator('#app')).toContainText('All preflight checks passed');
  await expect(iframe.getByRole('button', { name: 'Back: Preflight checks' })).toBeVisible();
  await iframe.getByRole('button', { name: 'Deploy', exact: true }).click();
}

async function waitForClusterUpdate(page: Page) {
  if (process.env.SKIP_CLUSTER_UPGRADE_CHECK !== 'true') {
    await expect(page.locator('.Modal-body')).toContainText('Cluster update in progress');
    await expect(page.locator('.Modal-body').getByText('Cluster update in progress')).not.toBeVisible({ timeout: 20 * 60 * 1000 });
  }
}

async function verifyUpgradeSuccess(page: Page) {
  await expect(page.locator('#app')).toContainText('Currently deployed version', { timeout: 5 * 60 * 1000 });
}
