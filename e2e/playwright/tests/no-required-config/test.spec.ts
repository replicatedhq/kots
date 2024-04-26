import { test, expect } from '@playwright/test';
import { login, uploadLicense } from '../shared';

const { execSync } = require("child_process");

test('no required config', async ({ page }) => {
  await login(page);
  await uploadLicense(page, expect);
  await expect(page.locator('#app')).toContainText('Configure No Required Config', { timeout: 15000 });
  const appStatus = execSync(`kubectl kots get apps -n ${process.env.NAMESPACE} | awk 'NR>1{print $2}'`).toString().trim();
  expect(appStatus).toBe('missing');
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Ready', { timeout: 30000 });
  await page.getByRole('link', { name: 'Version history' }).click();
  await expect(page.locator('#app')).toContainText('Currently deployed version');
});
