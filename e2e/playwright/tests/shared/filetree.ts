import { Page, Locator } from "@playwright/test";

export const filetreeSelectFile = async (page: Page, fileTree: Locator, file: string) => {
  await fileTree.getByTestId(file).scrollIntoViewIfNeeded();
  await fileTree.getByTestId(file).dispatchEvent('click');
  await page.waitForTimeout(500); // a small delay to ensure the ui has time to update
};
