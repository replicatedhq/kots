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
  validateDashboardInfo,
  validateDashboardAutomaticUpdates,
} from '../shared';

test('type=existing cluster, env=online, phase=new install, rbac=cluster admin', async ({ page }) => {
  test.setTimeout(30 * 60 * 1000); // 30 minutes

  deleteKurlConfigMap(constants.IS_AIRGAPPED);
  const registryCreds = getRegistryCredentials(constants.IS_AIRGAPPED, constants.IS_EXISTING_CLUSTER);

  // TODO NOW: uncomment this
  // installVeleroAWS(constants.VELERO_VERSION, constants.VELERO_AWS_PLUGIN_VERSION);
  await promoteVendorRelease(constants.VENDOR_INITIAL_RELEASE_SEQUENCE, constants.CHANNEL_ID, "1.0.0");

  await page.goto('/');
  await expect(page.getByTestId("build-version")).toHaveText(process.env.NEW_KOTS_VERSION!);

  await login(page);
  await uploadLicense(page, expect);

  await expect(page.locator("#app")).toContainText("Install in airgapped environment", { timeout: 15000 });
  await page.getByTestId("download-app-from-internet").click();

  await validateInitialConfig(page, expect);
  await validateClusterAdminInitialPreflights(page, expect);
  await validateDashboardInfo(page, expect);
  await validateDashboardAutomaticUpdates(page, expect);
});
