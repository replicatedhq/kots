import { Page, Expect } from '@playwright/test';

import { RegistryInfo } from './cli';

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

  await expect(page.getByTestId("airgap-upload-progress")).toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId("airgap-upload-progress-title")).toBeVisible();
  await expect(page.getByTestId("airgap-upload-progress-message")).toBeVisible();
  await expect(page.getByTestId("airgap-upload-progress-bar")).toBeVisible();
  await expect(page.getByTestId("airgap-upload-progress")).not.toBeVisible({ timeout: 60000 });
};
