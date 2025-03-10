import { test, expect } from '@playwright/test';
import * as constants from './constants';

import {
  INITIAL_VERSION_SMALL_BUNDLE_PATH,
  NEW_VERSION_SMALL_BUNDLE_PATH,
  INITIAL_VERSION_BUNDLE_PATH,
  NEW_VERSION_BUNDLE_PATH
} from '../shared/constants';

import {
  login,
  uploadLicense,
  downloadAirgapBundle,
  deleteKurlConfigMap,
  getRegistryInfo,
  validateUiAirgapInstall,
  validateSmallAirgapInitialConfig,
  validateSmallAirgapInitialPreflights,
  validateUiAirgapUpdate,
  validateCliAirgapUpdate,
  validateDashboardInfo,
  removeApp,
  removeKots,
  cliAirgapInstall,
  installVeleroHostPath,
  validateDashboardGraphs,
  updateConfig,
  validateIgnorePreflightsModal,
  validateCurrentVersionCard,
  validateCurrentClusterAdminPreflights,
  validateCurrentDeployLogs,
  validateConfigView,
  validateVersionHistoryRows,
  deployNewVersion,
  validateCurrentLicense,
  updateAirgappedLicense,
  validateUpdatedLicense,
  validateVersionDiff,
  validateSnapshotsHostPathConfig,
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
  logout
} from '../shared';

test('type=existing cluster, env=airgapped, phase=new install, rbac=cluster admin', async ({ page }) => {
  test.setTimeout(45 * 60 * 1000); // 45 minutes

  // Initial setup
  deleteKurlConfigMap();
  const registryInfo = getRegistryInfo(constants.IS_EXISTING_CLUSTER);

  // install Velero for snapshots
  await installVeleroHostPath(
    constants.VELERO_VERSION,
    constants.VELERO_AWS_PLUGIN_VERSION,
    registryInfo,
    constants.IS_AIRGAPPED
  );

  // download initial small airgap bundle for ui install
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.INITIAL_SMALL_BUNDLE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    INITIAL_VERSION_SMALL_BUNDLE_PATH
  );

  // download update small airgap bundle for ui update
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.UPDATE_SMALL_BUNDLE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    NEW_VERSION_SMALL_BUNDLE_PATH
  );

  // download initial airgap bundle
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_INITIAL_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    INITIAL_VERSION_BUNDLE_PATH
  );

  // download new airgap bundle
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_UPDATE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    NEW_VERSION_BUNDLE_PATH
  );

  // Login and license upload
  await page.goto('/');
  await expect(page.getByTestId("build-version")).toHaveText(process.env.NEW_KOTS_VERSION!);
  await login(page);
  await uploadLicense(page, expect);

  // Validate ui install and app updates
  await validateUiAirgapInstall(page, expect, registryInfo, constants.NAMESPACE, INITIAL_VERSION_SMALL_BUNDLE_PATH, constants.IS_EXISTING_CLUSTER);
  await validateSmallAirgapInitialConfig(page, expect);
  await validateSmallAirgapInitialPreflights(page, expect);
  await validateDashboardInfo(page, expect, constants.IS_AIRGAPPED);
  await validateUiAirgapUpdate(page, expect, NEW_VERSION_SMALL_BUNDLE_PATH);

  // Clean up UI install so we can test CLI install
  await logout(page, expect);
  removeApp(constants.NAMESPACE);
  removeKots(constants.NAMESPACE);

  // CLI airgap install
  cliAirgapInstall(
    registryInfo,
    INITIAL_VERSION_BUNDLE_PATH,
    `${process.env.TEST_PATH}/license.yaml`,
    `${process.env.TEST_PATH}/config.yaml`,
    constants.NAMESPACE,
    constants.IS_MINIMAL_RBAC
  );

  // Validate CLI install and app updates
  await page.waitForTimeout(5000);
  await page.reload();
  await expect(page.getByTestId("build-version")).toHaveText(process.env.NEW_KOTS_VERSION!);
  await login(page);
  await validateDashboardInfo(page, expect, constants.IS_AIRGAPPED);
  await validateDashboardGraphs(page, expect);
  await validateCliAirgapUpdate(
    page,
    expect,
    1,
    constants.IS_MINIMAL_RBAC,
    NEW_VERSION_BUNDLE_PATH,
    constants.NAMESPACE,
    constants.IS_EXISTING_CLUSTER,
    registryInfo
  );

  // Config update and version history checks
  await updateConfig(page, expect);
  await page.getByRole('button', { name: 'Deploy', exact: true }).first().click();
  await validateIgnorePreflightsModal(page, expect);
  await validateCurrentVersionCard(page, expect, 1);
  await validateCurrentClusterAdminPreflights(page, expect);
  await validateCurrentDeployLogs(page, expect);
  await validateConfigView(page, expect);
  await validateVersionHistoryRows(page, expect, constants.IS_AIRGAPPED);
  await deployNewVersion(page, expect, 2, 'Config Change', constants.IS_MINIMAL_RBAC);

  // License validation
  await validateCurrentLicense(page, expect, constants.CUSTOMER_NAME, constants.CHANNEL_NAME, constants.IS_AIRGAP_SUPPORTED, constants.IS_EC);
  await updateAirgappedLicense(page, expect, 'new-license.yaml');
  await validateUpdatedLicense(page, expect, 123, 3); // this is the value in the new license file
  await validateVersionDiff(page, expect, 3, 2);
  await deployNewVersion(page, expect, 3, 'License Change', constants.IS_MINIMAL_RBAC);

  // Snapshot validation
  await validateSnapshotsHostPathConfig(page, expect);
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
  await logout(page, expect);
});
