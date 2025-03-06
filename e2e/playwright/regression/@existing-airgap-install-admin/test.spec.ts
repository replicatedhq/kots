import { test, expect } from '@playwright/test';
import * as constants from './constants';

import {
  SMALL_BUNDLE_PATH,
  INITIAL_VERSION_BUNDLE_PATH,
  NEW_VERSION_BUNDLE_PATH
} from '../shared/constants';

import {
  login,
  uploadLicense,
  downloadAirgapBundle,
  deleteKurlConfigMap,
  getRegistryInfo,
  uiAirgapInstall,
  validateSmallAirgapInitialConfig,
  validateSmallAirgapInitialPreflights,
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
  deployVersion,
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

test('type=existing cluster, env=airgapped, phase=new install, rbac=cluster admin', async ({ page }) => {
  test.setTimeout(45 * 60 * 1000); // 45 minutes

  // download small airgap bundle for ui upload
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.SMALL_BUNDLE_CHANNEL_RELEASE_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    SMALL_BUNDLE_PATH,
    constants.SSH_TO_AIRGAPPED_INSTANCE
  );

  // download initial airgap bundle
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_INITIAL_CHANNEL_RELEASE_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    INITIAL_VERSION_BUNDLE_PATH,
    constants.SSH_TO_AIRGAPPED_INSTANCE
  );

  // download new airgap bundle
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_UPDATE_CHANNEL_RELEASE_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    NEW_VERSION_BUNDLE_PATH,
    constants.SSH_TO_AIRGAPPED_INSTANCE
  );

  deleteKurlConfigMap(constants.SSH_TO_AIRGAPPED_INSTANCE);
  const registryInfo = getRegistryInfo(constants.IS_EXISTING_CLUSTER, constants.SSH_TO_AIRGAPPED_INSTANCE);

  await page.goto('/');
  await expect(page.getByTestId("build-version")).toHaveText(process.env.NEW_KOTS_VERSION!);

  await login(page);
  await uploadLicense(page, expect);

  await uiAirgapInstall(page, expect, registryInfo, constants.NAMESPACE, SMALL_BUNDLE_PATH, constants.IS_EXISTING_CLUSTER);
  await validateSmallAirgapInitialConfig(page, expect);
  await validateSmallAirgapInitialPreflights(page, expect);

  await validateInitialConfig(page, expect);
  await validateClusterAdminInitialPreflights(page, expect);
  await validateDashboardInfo(page, expect);
  await validateDashboardAutomaticUpdates(page, expect);
  await validateDashboardGraphs(page, expect);
  await updateConfig(page, expect);

  await page.getByRole('button', { name: 'Deploy', exact: true }).first().click();
  await validateIgnorePreflightsModal(page, expect);
  await validateVersionHistoryAutomaticUpdates(page, expect);

  await validateCurrentVersionCard(page, expect, "1.0.0", 0);
  await validateCurrentReleaseNotes(page, expect, "release notes - updates");
  await validateCurrentClusterAdminPreflights(page, expect);
  await validateCurrentDeployLogs(page, expect);
  await validateConfigView(page, expect);
  await validateVersionHistoryRows(page, expect, true);
  await deployVersion(page, expect, 0, 1, 'Config Change', false);

  await validateCurrentLicense(page, expect, constants.CUSTOMER_NAME, constants.CHANNEL_NAME, constants.IS_AIRGAP_SUPPORTED, constants.IS_EC);
  const newIntEntitlement = await updateOnlineLicense(page, constants.CUSTOMER_ID, constants.CUSTOMER_NAME, constants.CHANNEL_ID, constants.IS_AIRGAP_SUPPORTED, constants.IS_EC);
  await validateUpdatedLicense(page, expect, newIntEntitlement);

  await validateVersionDiff(page, expect, 2, 1);
  await deployVersion(page, expect, 0, 2, 'License Change', false);

  await validateSnapshotsAWSConfig(page, expect);
  await validateAutomaticFullSnapshots(page, expect);
  await validateAutomaticPartialSnapshots(page, expect);
  await createAppSnapshot(page, expect);
  await rollbackToVersion(page, expect, 1, 1);
  await restoreAppSnapshot(page, expect, 0, 2, true);
  await deleteAppSnapshot(page, expect);
  await createFullSnapshot(page, expect);
  await rollbackToVersion(page, expect, 1, 1);
  await restoreFullSnapshot(page, expect, 0, 2, true, constants.IS_AIRGAPPED);
  await deleteFullSnapshot(page, expect);

  await validateViewFiles(page, expect, constants.CHANNEL_ID, constants.CHANNEL_NAME, constants.CUSTOMER_NAME, constants.LICENSE_ID, constants.IS_AIRGAPPED, registryInfo);
  await updateRegistrySettings(page, expect, registryInfo, false);
  await validateCheckForUpdates(page, expect, constants.CHANNEL_ID, constants.VENDOR_UPDATE_CHANNEL_RELEASE_SEQUENCE, 4, false);
  await validateDuplicateLicenseUpload(page, expect);
  await logout(page, expect);
});
