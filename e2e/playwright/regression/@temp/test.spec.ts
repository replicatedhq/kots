import { test, expect } from '@playwright/test';
import * as constants from './constants';

import {
  login,
  uploadLicense,
  deleteKurlConfigMap,
  getRegistryCredentials,
  installVeleroAWS,
  promoteVendorRelease,
  validateInitialConfig,
  validateClusterAdminInitialPreflights,
  addSnapshotsRBAC,
} from '../shared';

test('type=existing cluster, env=online, phase=new install, rbac=cluster admin', async ({ page }) => {
  test.setTimeout(30 * 60 * 1000); // 30 minutes

  deleteKurlConfigMap(constants.IS_AIRGAPPED);
  const registryCreds = getRegistryCredentials(constants.IS_AIRGAPPED, constants.IS_EXISTING_CLUSTER);

  installVeleroAWS(constants.VELERO_VERSION, constants.VELERO_AWS_PLUGIN_VERSION);
  await promoteVendorRelease(constants.VENDOR_INITIAL_RELEASE_SEQUENCE, constants.CHANNEL_ID, "1.0.0");

  // TODO NOW: get testNewKotsVersion from whatever runs playwright
  const testNewKotsVersion = "alpha";
  await expect(page.getByTestId("build-version")).toHaveText(testNewKotsVersion);

  await login(page);
  await uploadLicense(page, expect);

  await expect(page.locator("#app")).toContainText("Install in airgapped environment", { timeout: 15000 });
  await page.getByTestId("download-app-from-internet").click();

  await validateInitialConfig(page, expect);
  await validateClusterAdminInitialPreflights(page, expect);

  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();

  const configureSnapshotsModal = page.getByTestId("configure-snapshots-modal");
  await expect(configureSnapshotsModal).toBeVisible({ timeout: 10000 });
  await configureSnapshotsModal.getByText("I've already installed Velero").click();
  await expect(configureSnapshotsModal.getByTestId("ensure-permissions-command")).toContainText('kubectl kots velero ensure-permissions --namespace default --velero-namespace <velero-namespace>');
  await configureSnapshotsModal.getByRole('button', { name: 'Ok, got it!' }).click();
  await expect(configureSnapshotsModal).not.toBeVisible();

  await addSnapshotsRBAC(page, expect, constants.NAMESPACE);
});
