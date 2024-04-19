import { test, expect } from '@playwright/test';

test('no required config', async ({ page }) => {
  await page.goto('/');
  await page.getByPlaceholder('password').click();
  await page.getByPlaceholder('password').fill('password');
  await page.getByRole('button', { name: 'Log in' }).click();
  await page.setInputFiles('input[type="file"][accept="application/x-yaml,.yaml,.yml,.rli"]', 'tests/no-required-config/license.yaml')
  await page.getByRole('button', { name: 'Upload license' }).click();
  await expect(page.locator('#app')).toContainText('Installing your license');
  await expect(page.locator('#app')).toContainText('Configure No Required Config', { timeout: 15000 });
  const appStatus = require('child_process').execSync(`kubectl kots get apps -n ${process.env.NAMESPACE} | awk 'NR>1{print $2}'`).toString().trim();
  expect(appStatus).toBe('missing');
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Ready');
  await page.getByRole('link', { name: 'Version history' }).click();
  await expect(page.locator('#app')).toContainText('Currently deployed version');
});
