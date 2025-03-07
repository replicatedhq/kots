import { Page, Expect } from '@playwright/test';

import { RegistryInfo } from './cli';
import {
  validateCurrentVersionCard,
  validateReleaseNotesModal,
  deployNewVersion
} from './version-history';

export const uiAirgapInstall = async (
  page: Page,
  expect: Expect,
  registryInfo: RegistryInfo,
  namespace: string,
  airgapBundlePath: string,
  isExistingCluster: boolean
) => {
  await expect(page.locator("#app")).toContainText("Install in airgapped environment", { timeout: 15000 });

  if (isExistingCluster) {
    await page.getByTestId("airgap-registry-hostname").click();
    await page.getByTestId("airgap-registry-hostname").fill(registryInfo.ip);
    await page.getByTestId("airgap-registry-username").click();
    await page.getByTestId("airgap-registry-username").fill(registryInfo.username);
    await page.getByTestId("airgap-registry-password").click();
    await page.getByTestId("airgap-registry-password").fill(registryInfo.password);
    await page.getByTestId("airgap-registry-namespace").click();
    await page.getByTestId("airgap-registry-namespace").fill(namespace);
  }

  await page.setInputFiles('[data-testid="airgap-bundle-drop-zone"] input', airgapBundlePath);
  await page.getByTestId("upload-airgap-bundle-button").click();

  const airgapUploadProgress = page.getByTestId("airgap-upload-progress");
  await expect(airgapUploadProgress).toBeVisible({ timeout: 15000 });
  await expect(airgapUploadProgress.getByTestId("processing-images-progress-title")).toBeVisible();
  await expect(airgapUploadProgress.getByTestId("processing-images-progress-message")).toBeVisible();
  await expect(airgapUploadProgress.getByTestId("processing-images-progress-bar")).toBeVisible();
  await expect(airgapUploadProgress).not.toBeVisible({ timeout: 60000 });
};

export const uiAirgapUpdate = async (
  page: Page,
  expect: Expect,
  airgapBundlePath: string,
) => {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible({ timeout: 15000 });
  await expect(updatesCard).toContainText("Application up to date.");

  // this is the version label in the airgap.yaml file of the initial small bundle
  await validateCurrentVersionCard(page, expect, "0.1.4", 0);

  await page.setInputFiles('[data-testid="airgap-bundle-drop-zone"] input', airgapBundlePath);
  const airgapUploadProgress = page.getByTestId("airgap-upload-progress");
  await expect(airgapUploadProgress).toBeVisible({ timeout: 15000 });
  await expect(airgapUploadProgress.getByTestId("airgap-upload-progress-title")).toBeVisible();
  await expect(airgapUploadProgress.getByTestId("airgap-upload-progress-bar")).toBeVisible();

  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).toBeVisible({ timeout: 45000 });
  await expect(updateRow).toContainText('Airgap Update');
  await expect(updateRow).toContainText('0.1.5'); // this is the version label in the airgap.yaml file of the new small bundle

  const preflightChecksLoader = updateRow.getByTestId('preflight-checks-loader');
  await expect(preflightChecksLoader).toBeVisible();
  await expect(preflightChecksLoader).not.toBeVisible({ timeout: 120000 });

  await updateRow.getByTestId('release-notes-icon').click();
  await validateReleaseNotesModal(page, expect, "release notes - updates");

  // minimal rbac is false because we uploaded the initial bundle via the ui.
  // in airgap, minimal rbac is only detected if the bundle is passed to cli install.
  await deployNewVersion(page, expect, 1, 'Airgap Update', false);
};

