import { Page, Expect } from '@playwright/test';

export const uploadLicense = async (page: Page, expect: Expect, licenseFile = "license.yaml") => {
  await page.setInputFiles('input[type="file"]', `${process.env.TEST_PATH}/${licenseFile}`);
  await page.getByRole('button', { name: 'Upload license' }).click();
  await expect(page.locator('#app')).toContainText('Installing your license');
};
