import { Page, Expect } from '@playwright/test';

import { RegistryInfo } from './cli';
import { APP_SLUG } from './constants';
import { deployNewVersion } from './version-history';
import { validateRegistryChangeKustomization } from './view-files';

export const updateRegistrySettings = async (page: Page, expect: Expect, registryInfo: RegistryInfo, expectedSequence: number, isMinimalRBAC: boolean) => {
  await page.getByRole('link', { name: 'Registry settings', exact: true }).click();

  const card = page.getByTestId('airgap-registry-settings-card');
  await expect(card).toBeVisible({ timeout: 15000 });

  await card.getByTestId("airgap-registry-hostname").click();
  await card.getByTestId("airgap-registry-hostname").fill(registryInfo.ip);
  await card.getByTestId("airgap-registry-username").click();
  await card.getByTestId("airgap-registry-username").fill(registryInfo.username);
  await card.getByTestId("airgap-registry-password").click();
  await card.getByTestId("airgap-registry-password").fill(registryInfo.password);

  const testConnectionBox = card.getByTestId("test-connection-box");
  await testConnectionBox.getByTestId("test-connection-button").click();
  await expect(testConnectionBox).toContainText('Success!', { timeout: 30000 });

  await card.getByTestId("airgap-registry-namespace").click();
  await card.getByTestId("airgap-registry-namespace").fill(APP_SLUG);

  await expect(card.getByTestId("disable-pushing-images-checkbox")).toBeVisible();
  await expect(card.getByTestId("disable-pushing-images-checkbox")).not.toBeChecked();

  await card.getByRole('button', { name: 'Save changes', exact: true }).click();

  const progress = card.getByTestId("airgap-registry-settings-progress");
  await expect(progress.locator('.Loader')).toBeVisible({ timeout: 30000 });
  await expect(progress.getByTestId("progress-text")).toContainText('ing', { timeout: 30000 });
  await expect(card.getByTestId("airgap-registry-settings-progress")).not.toBeVisible({ timeout: 240000 });

  await page.reload();
  await deployNewVersion(page, expect, expectedSequence, 'Registry Change', isMinimalRBAC);

  await validateRegistryChangeKustomization(page, expect, registryInfo);
};
