import { Page, Expect } from '@playwright/test';

export const appIsReady = async (page: Page, expect: Expect, timeout: number = 30 * 1000) => {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Dashboard" }).click();
  await expect(page.getByTestId("page-dashboard").getByTestId("app-status-status")).toContainText("Ready", { timeout });
};
