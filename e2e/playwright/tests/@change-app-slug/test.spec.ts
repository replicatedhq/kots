import { test, expect } from '@playwright/test';
import { login, uploadLicense } from '../shared';

const aliasSlug = 'change-app-slug-alias';

test('install an original-slug license using the current server alias', async ({ page }) => {
  test.setTimeout(5 * 60 * 1000);
  await login(page);
  await uploadLicense(page, expect, '../@change-license/change-app-slug-orignal.yaml');
  // Online installation fetches the latest license before creating the app.
  await expect(page).toHaveURL(new RegExp(`/${aliasSlug}/config$`), { timeout: 30000 });
  await expect(page.locator('#app')).toContainText('Configure App Name', { timeout: 30000 });
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Results', { timeout: 60000 });
  await page.getByRole('button', { name: 'Deploy', exact: true }).click();
  const deployAnyway = page.getByRole('button', { name: 'Deploy anyway' });
  await deployAnyway.waitFor({ state: 'visible', timeout: 5000 }).catch(() => {});
  if (await deployAnyway.isVisible()) {
    await deployAnyway.click();
  }
  await expect(page).toHaveURL(new RegExp(`/app/${aliasSlug}$`), { timeout: 30000 });
  await expect(page.locator('#app')).toContainText('Ready', { timeout: 60000 });
  const apps = await page.evaluate(async () => {
    const response = await fetch('/api/v1/apps', { credentials: 'include' });
    if (!response.ok) throw new Error(`Failed to list apps: ${response.status}`);
    return response.json();
  });
  expect(apps.apps).toHaveLength(1);
  expect(apps.apps[0].slug).toBe(aliasSlug);
  expect(apps.apps[0].upstreamUri).toBe(`replicated://${aliasSlug}`);

  await page.getByRole('link', { name: 'License', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/app/${aliasSlug}/license$`));
});
