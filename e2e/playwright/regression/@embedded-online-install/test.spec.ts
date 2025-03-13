import { test, expect } from '@playwright/test';
import * as constants from './constants';

import {
  login,
  uploadLicense,
  getRegistryInfo,
  promoteRelease,
  resetPassword,
  validateInitialConfig,
  validateClusterAdminInitialPreflights,
  joinWorkerNode,
  validateDashboardInfo,
  validateDashboardAutomaticUpdates,
  validateDashboardGraphs,
  updateConfig,
  validateIgnorePreflightsModal,
  validateVersionHistoryAutomaticUpdates,
  validateCurrentVersionCard,
  validateCurrentClusterAdminPreflights,
  validateCurrentDeployLogs,
  validateConfigView,
  validateVersionHistoryRows,
  deployNewVersion,
  validateCurrentLicense,
  updateOnlineLicense,
  validateUpdatedLicense,
  validateVersionDiff,
  configureSnapshotsAWSInstanceRole,
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
  validateClusterManagement,
  validateIdentityService,
  logout
} from '../shared';

test('type=embedded cluster, env=online, phase=new install, rbac=cluster admin', async ({ page }) => {
  test.setTimeout(30 * 60 * 1000); // 30 minutes

  // Initial setup
  resetPassword(constants.NAMESPACE);
  const registryInfo = getRegistryInfo(constants.IS_EXISTING_CLUSTER);
  await promoteRelease(constants.VENDOR_INITIAL_CHANNEL_SEQUENCE, constants.CHANNEL_ID, "1.0.0");

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
  await joinWorkerNode(page, expect, constants.IS_AIRGAPPED); // runs in the background
  await validateDashboardInfo(page, expect, constants.IS_AIRGAPPED);
  await validateDashboardAutomaticUpdates(page, expect);
  await validateDashboardGraphs(page, expect);
  await validateCheckForUpdates(page, expect, constants.CHANNEL_ID, constants.VENDOR_UPDATE_CHANNEL_SEQUENCE, 1, constants.IS_MINIMAL_RBAC);

  // Config update and version history checks
  await updateConfig(page, expect);
  await page.getByRole('button', { name: 'Deploy', exact: true }).first().click();
  await validateIgnorePreflightsModal(page, expect);
  await validateVersionHistoryAutomaticUpdates(page, expect);
  await validateCurrentVersionCard(page, expect, 1);
  await validateCurrentClusterAdminPreflights(page, expect);
  await validateCurrentDeployLogs(page, expect);
  await validateConfigView(page, expect);
  await validateVersionHistoryRows(page, expect, constants.IS_AIRGAPPED);
  await deployNewVersion(page, expect, 2, 'Config Change', constants.IS_MINIMAL_RBAC);

  // License validation
  await validateCurrentLicense(page, expect, constants.CUSTOMER_NAME, constants.CHANNEL_NAME, constants.IS_AIRGAP_SUPPORTED, constants.IS_EC);
  const newIntEntitlement = await updateOnlineLicense(page, constants.CUSTOMER_ID, constants.CUSTOMER_NAME, constants.CHANNEL_ID, constants.IS_AIRGAP_SUPPORTED, constants.IS_EC);
  await validateUpdatedLicense(page, expect, newIntEntitlement, 3);
  await validateVersionDiff(page, expect, 3, 2);
  await deployNewVersion(page, expect, 3, 'License Change', constants.IS_MINIMAL_RBAC);

  // Snapshot validation
  await configureSnapshotsAWSInstanceRole(page, expect);
  await validateAutomaticFullSnapshots(page, expect);
  await validateAutomaticPartialSnapshots(page, expect);

  // App snapshot workflow
  await createAppSnapshot(page, expect);
  await rollbackToVersion(page, expect, 1, 2);
  await restoreAppSnapshot(page, expect, 0, 3, true, constants.IS_AIRGAPPED);
  await deleteAppSnapshot(page, expect);

  // Full snapshot workflow
  await createFullSnapshot(page, expect);
  await rollbackToVersion(page, expect, 1, 2);
  await restoreFullSnapshot(page, expect, 0, 3, true, constants.IS_AIRGAPPED);
  await deleteFullSnapshot(page, expect);

  // Other validation
  await validateViewFiles(page, expect, constants.CHANNEL_ID, constants.CHANNEL_NAME, constants.CUSTOMER_NAME, constants.LICENSE_ID, constants.IS_AIRGAPPED, registryInfo);
  await updateRegistrySettings(page, expect, registryInfo, 4, constants.IS_MINIMAL_RBAC);
  await validateClusterManagement(page, expect);
  await validateIdentityService(page, expect, constants.NAMESPACE, constants.IS_AIRGAPPED);
  await logout(page, expect);
});
