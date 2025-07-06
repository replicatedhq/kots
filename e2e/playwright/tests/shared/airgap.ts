import { Page, Expect, Locator } from '@playwright/test';

export const airgapInstall = async (page: Page, expect: Expect, host: string, username: string, password: string, namespace: string, airgapBundlePath: string, timeout: number = 2 * 60 * 1000) => {
  await expect(page.locator("#app")).toContainText("Install in airgapped environment", { timeout: 15000 });
  await page.getByTestId("airgap-registry-hostname").click();
  await page.getByTestId("airgap-registry-hostname").fill(host);
  await page.getByTestId("airgap-registry-username").click();
  await page.getByTestId("airgap-registry-username").fill(username);
  await page.getByTestId("airgap-registry-password").click();
  await page.getByTestId("airgap-registry-password").fill(password);
  await page.getByTestId("airgap-registry-namespace").click();
  await page.getByTestId("airgap-registry-namespace").fill(namespace);
  await page.setInputFiles('[data-testid="airgap-bundle-drop-zone"] input', airgapBundlePath);
  await page.getByTestId("upload-airgap-bundle-button").click();
  await expect(page.getByTestId("airgap-upload-progress")).toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId("airgap-upload-progress")).not.toBeVisible({ timeout });
};

export const airgapOnlineInstall = async (page: Page, expect: Expect) => {
  await expect(page.locator("#app")).toContainText("Install in airgapped environment", { timeout: 15000 });
  await page.getByText(/download .* from the Internet/).click();
  await expect(page.locator("#app")).toContainText("Installing your license");
};

export const airgapInstallErrorMessage = (page: Page): Locator => {
  return page.getByTestId("airgap-bundle-upload-error");
};

export const airgapUpdate = async (page: Page, expect: Expect, airgapBundlePath: string) => {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Version history" }).click();
  await expect(page.locator("#app")).toContainText("Currently deployed version", { timeout: 15000 });
  await page.setInputFiles('[data-testid="airgap-bundle-drop-zone"] input', airgapBundlePath);
};

export const airgapUpdateErrorMessage = (page: Page): Locator => {
  return page.getByTestId("airgap-bundle-upload-error");
};
