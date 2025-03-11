import { Page, Expect } from '@playwright/test';

export const appIsReady = async (page: Page, expect: Expect, timeout: number = 2 * 60 * 1000) => { // 2 minutes
  await page.getByTestId("console-subnav").getByRole("link", { name: "Dashboard" }).click();
  await expect(page.getByTestId("page-dashboard").getByTestId("dashboard-app-status")).toContainText("Ready", { timeout });
};
