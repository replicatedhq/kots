import { Page, Expect } from '@playwright/test';

export const validateInitialConfig = async (page: Page, expect: Expect) => {
  const sidebar = page.getByTestId('config-sidebar-wrapper');
  await expect(sidebar).toContainText('Nginx Config', { timeout: 15000 });
  await expect(sidebar).toContainText('Nginx port');
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar).toContainText('a bool field');
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar).not.toContainText('a bool field');
  
  const configArea = page.getByTestId('config-area');
  await expect(configArea).toContainText('Nginx Config');
  await expect(configArea).toContainText('Nginx port');
  await configArea.getByText('a bool field').click();

  await configArea.locator('#a_required_text-group').getByRole('textbox').click();
  await configArea.locator('#a_required_text-group').getByRole('textbox').fill('i filled this because it is required');
  await configArea.getByLabel('Check to include helm chart').check();
  await page.getByRole('button', { name: 'Continue' }).click();
};
