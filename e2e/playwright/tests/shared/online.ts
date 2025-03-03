import { Page, Expect } from '@playwright/test';

export const onlineCheckForUpdates = async (page: Page, expect: Expect) => {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Version history" }).click();
  await page.getByTestId("check-for-update-button").click();
};
