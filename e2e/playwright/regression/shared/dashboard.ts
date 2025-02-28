import { Page, Expect } from '@playwright/test';

export const validateDashboardInfo = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Application', { exact: true }).click();
  await page.getByRole('link', { name: 'Dashboard', exact: true }).click();

  await expect(page.getByTestId("dashboard-app-icon")).toBeVisible();
  await expect(page.getByTestId("dashboard-app-name")).toBeVisible();
  await expect(page.getByTestId("dashboard-app-status")).toHaveText("Ready", { timeout: 210000 });
  await expect(page.getByTestId("dashboard-edit-config")).toBeVisible();
  await expect(page.getByTestId("dashboard-app-link")).toBeVisible();

  const versionCard = page.getByTestId("dashboard-version-card");
  await expect(versionCard).toBeVisible();
  await expect(versionCard.getByTestId("current-version-status")).toHaveText('Currently deployed version', { timeout: 30000 });
  await expect(versionCard.getByText("See all versions")).toBeVisible();

  const licenseCard = page.getByTestId("dashboard-license-card");
  await expect(licenseCard).toBeVisible();
  await expect(licenseCard.getByText("Sync license")).toBeVisible();
  await expect(licenseCard.getByText("See license details")).toBeVisible();

  const snapshotsCard = page.getByTestId("dashboard-snapshots-card");
  await expect(snapshotsCard).toBeVisible();
  await expect(snapshotsCard.getByText("Snapshot settings")).toBeVisible();
  await expect(snapshotsCard.getByText("Start snapshot")).toBeVisible();
  await expect(snapshotsCard.getByText("See all snapshots")).toBeVisible();
};
