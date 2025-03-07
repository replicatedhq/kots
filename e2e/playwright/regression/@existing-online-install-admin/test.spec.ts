import { test, expect } from '@playwright/test';
import * as constants from './constants';

import {
  login,
  uploadLicense,
  deleteKurlConfigMap,
  getRegistryInfo,
  installVeleroAWS,
  promoteRelease,
  validateInitialConfig,
  validateClusterAdminInitialPreflights,
  validateDashboardInfo,
  validateDashboardAutomaticUpdates,
  validateDashboardGraphs,
  updateConfig,
  validateIgnorePreflightsModal,
  validateVersionHistoryAutomaticUpdates,
  validateCurrentVersionCard,
  validateCurrentReleaseNotes,
  validateCurrentClusterAdminPreflights,
  validateCurrentDeployLogs,
  validateConfigView,
  validateVersionHistoryRows,
  deployNewVersion,
  validateCurrentLicense,
  updateOnlineLicense,
  validateUpdatedLicense,
  validateVersionDiff,
  validateSnapshotsAWSConfig,
  validateAutomaticFullSnapshots,
  validateAutomaticPartialSnapshots,
  createAppSnapshot,
  rollbackToVersion,
  restoreAppSnapshot,
  deleteAppSnapshot,
  createFullSnapshot,
  restoreFullSnapshot,
  deleteFullSnapshot,
  validateViewFiles,
  updateRegistrySettings,
  validateCheckForUpdates,
  validateDuplicateLicenseUpload,
  logout
} from '../shared';

test('type=existing cluster, env=online, phase=new install, rbac=cluster admin', async ({ page }) => {
  test.setTimeout(30 * 60 * 1000); // 30 minutes

  // Initial setup
  deleteKurlConfigMap(constants.IS_AIRGAPPED);
  const registryInfo = getRegistryInfo(constants.IS_AIRGAPPED, constants.IS_EXISTING_CLUSTER);
  installVeleroAWS(constants.VELERO_VERSION, constants.VELERO_AWS_PLUGIN_VERSION);
  await promoteRelease(constants.VENDOR_INITIAL_RELEASE_SEQUENCE, constants.CHANNEL_ID, "1.0.0");

  // Login and install
  await page.goto('/');
  await expect(page.getByTestId("build-version")).toHaveText(process.env.NEW_KOTS_VERSION!);
  await login(page);
  await uploadLicense(page, expect);
  await expect(page.locator("#app")).toContainText("Install in airgapped environment", { timeout: 15000 });
  await page.getByTestId("download-app-from-internet").click();

  // Validate install and app updates
  await validateInitialConfig(page, expect);
  await validateClusterAdminInitialPreflights(page, expect);
  await validateDashboardInfo(page, expect);
  await validateDashboardAutomaticUpdates(page, expect);
  await validateDashboardGraphs(page, expect);
  await validateCheckForUpdates(page, expect, constants.CHANNEL_ID, constants.VENDOR_UPDATE_RELEASE_SEQUENCE, 1, false);

  // Config update and version history checks
  await updateConfig(page, expect);
  await page.getByRole('button', { name: 'Deploy', exact: true }).first().click();
  await validateIgnorePreflightsModal(page, expect);
  await validateVersionHistoryAutomaticUpdates(page, expect);
  await validateCurrentVersionCard(page, expect, "1.0.0", 1);
  await validateCurrentReleaseNotes(page, expect, "release notes - updates");
  await validateCurrentClusterAdminPreflights(page, expect);
  await validateCurrentDeployLogs(page, expect);
  await validateConfigView(page, expect);
  await validateVersionHistoryRows(page, expect, constants.IS_AIRGAPPED);
  await deployNewVersion(page, expect, 2, 'Config Change', false);

  // License validation
  await validateCurrentLicense(page, expect, constants.CUSTOMER_NAME, constants.CHANNEL_NAME, constants.IS_AIRGAP_SUPPORTED, constants.IS_EC);
  const newIntEntitlement = await updateOnlineLicense(page, constants.CUSTOMER_ID, constants.CUSTOMER_NAME, constants.CHANNEL_ID, constants.IS_AIRGAP_SUPPORTED, constants.IS_EC);
  await validateUpdatedLicense(page, expect, newIntEntitlement);
  await validateVersionDiff(page, expect, 3, 2);
  await deployNewVersion(page, expect, 3, 'License Change', false);

  // Snapshot validation
  await validateSnapshotsAWSConfig(page, expect);
  await validateAutomaticFullSnapshots(page, expect);
  await validateAutomaticPartialSnapshots(page, expect);

  // App snapshot workflow
  await createAppSnapshot(page, expect);
  await rollbackToVersion(page, expect, 1, 2);
  await restoreAppSnapshot(page, expect, 0, 3, true);
  await deleteAppSnapshot(page, expect);

  // Full snapshot workflow
  await createFullSnapshot(page, expect);
  await rollbackToVersion(page, expect, 1, 2);
  await restoreFullSnapshot(page, expect, 0, 3, true, constants.IS_AIRGAPPED);
  await deleteFullSnapshot(page, expect);

  // Other validation
  await validateViewFiles(page, expect, constants.CHANNEL_ID, constants.CHANNEL_NAME, constants.CUSTOMER_NAME, constants.LICENSE_ID, constants.IS_AIRGAPPED, registryInfo);
  await updateRegistrySettings(page, expect, registryInfo, 4, false);
  await validateDuplicateLicenseUpload(page, expect);
  await logout(page, expect);
});
