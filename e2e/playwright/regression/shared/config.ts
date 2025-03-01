import { Page, Expect } from '@playwright/test';

import * as uuid from "uuid";

export const validateInitialConfig = async (page: Page, expect: Expect) => {
  const sidebar = page.getByTestId('config-sidebar-wrapper');
  await expect(sidebar).toContainText('Nginx Config', { timeout: 15000 });
  await expect(sidebar).toContainText('Nginx port');
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).toBeVisible();
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).not.toBeVisible();

  const configArea = page.getByTestId('config-area');
  await expect(configArea).toContainText('Nginx Config');
  await expect(configArea).toContainText('Nginx port');
  await configArea.getByText('a bool field').click();

  await configArea.locator('#a_required_text-group').getByRole('textbox').click();
  await configArea.locator('#a_required_text-group').getByRole('textbox').fill('i filled this because it is required');
  await configArea.getByLabel('Check to include helm chart').check();
  await page.getByRole('button', { name: 'Continue' }).click();
};

export const updateConfig = async (page: Page, expect: Expect) => {
  await page.getByRole('link', { name: 'Config', exact: true }).click();
  const sidebar = page.getByTestId('config-sidebar-wrapper');
  await expect(sidebar).toContainText('Nginx Config', { timeout: 15000 });
  await expect(sidebar).toContainText('Nginx port');
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).toBeVisible();
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).not.toBeVisible();

  const configArea = page.getByTestId('config-area');
  await expect(configArea).toContainText('Nginx Config');
  await expect(configArea).toContainText('Nginx port');
  await configArea.getByText('a bool field').click();

  await configArea.locator('#a_text-group').getByRole('textbox').click();
  await configArea.locator('#a_text-group').getByRole('textbox').fill('a new value');

  await configArea.locator('#a_textarea-group').getByRole('textbox').click();
  await configArea.locator('#a_textarea-group').getByRole('textbox').fill('a new value for textarea');

  await configArea.locator('#a_required_text-group').getByRole('textbox').click();
  await configArea.locator('#a_required_text-group').getByRole('textbox').fill("i want to update this field - " + uuid.v4());

  await page.waitForTimeout(5000);
  await page.getByRole('button', { name: 'Save config' }).click();
  await expect(page.getByTestId('config-next-step-modal')).toBeVisible({ timeout: 15000 });
  await page.getByRole('button', { name: 'Go to updated version' }).click();
};
