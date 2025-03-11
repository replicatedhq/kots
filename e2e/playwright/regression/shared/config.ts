import { Page, Expect } from '@playwright/test';

import * as uuid from "uuid";

export const validateInitialConfig = async (page: Page, expect: Expect) => {
  const sidebar = page.getByTestId('config-sidebar-wrapper');
  await expect(sidebar).toBeVisible({ timeout: 15000 });
  await expect(sidebar).toContainText('Nginx Config');
  await expect(sidebar).toContainText('Nginx port');
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).toBeVisible();
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).not.toBeVisible();

  const configArea = page.getByTestId('config-area');
  await expect(configArea).toBeVisible();
  await expect(configArea).toContainText('Nginx Config');
  await expect(configArea).toContainText('Nginx port');
  await configArea.getByText('a bool field').click();

  await configArea.locator('#a_required_text-group').getByRole('textbox').click();
  await configArea.locator('#a_required_text-group').getByRole('textbox').fill('i filled this because it is required');
  await configArea.getByLabel('Check to include helm chart').check();
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(configArea).not.toBeVisible({ timeout: 30000 });
};

export const updateConfig = async (page: Page, expect: Expect) => {
  await page.getByRole('link', { name: 'Config', exact: true }).click();
  const sidebar = page.getByTestId('config-sidebar-wrapper');
  await expect(sidebar).toBeVisible({ timeout: 15000 });
  await expect(sidebar).toContainText('Nginx Config');
  await expect(sidebar).toContainText('Nginx port');
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).toBeVisible();
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('a bool field')).not.toBeVisible();

  const configArea = page.getByTestId('config-area');
  await expect(configArea).toBeVisible();
  await expect(configArea).toContainText('Nginx Config');
  await expect(configArea).toContainText('Nginx port');
  await configArea.getByText('a bool field').click();

  await configArea.locator('#a_text-group').getByRole('textbox').click();
  await configArea.locator('#a_text-group').getByRole('textbox').fill('a new value');

  await configArea.locator('#a_textarea-group').getByRole('textbox').click();
  await configArea.locator('#a_textarea-group').getByRole('textbox').fill('a new value for textarea');

  await configArea.locator('#a_required_text-group').getByRole('textbox').click();
  await configArea.locator('#a_required_text-group').getByRole('textbox').fill("i want to update this field - " + uuid.v4());

  await page.waitForTimeout(5000); // config page can take a bit to re-render
  await page.getByRole('button', { name: 'Save config' }).click();

  const nextStepModal = page.getByTestId('config-next-step-modal');
  await expect(nextStepModal).toBeVisible({ timeout: 30000 });
  await page.getByRole('button', { name: 'Go to updated version' }).click();
  await expect(nextStepModal).not.toBeVisible();
};

export const validateConfigView = async (page: Page, expect: Expect) => {
  await page.getByRole('link', { name: 'Config', exact: true }).click();
  const sidebar = page.getByTestId('config-sidebar-wrapper');
  await expect(sidebar).toBeVisible({ timeout: 15000 });

  const configArea = page.getByTestId('config-area');
  await expect(configArea).toBeVisible();
  await expect(configArea.getByTestId('config-info-current')).toBeVisible();

  await configArea.getByTestId('config-info-edit-latest').click();
  await page.waitForTimeout(1000);
  await expect(sidebar).toBeVisible({ timeout: 15000 });
  await expect(configArea).toBeVisible();
  await expect(configArea.getByTestId('config-info-newer')).toBeVisible();
};

export const validateSmallAirgapInitialConfig = async (page: Page, expect: Expect) => {
  const sidebar = page.getByTestId('config-sidebar-wrapper');
  await expect(sidebar).toBeVisible({ timeout: 15000 });

  await expect(sidebar.getByText('Use Ingress?')).toBeVisible();
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('Use Ingress?')).not.toBeVisible();
  await sidebar.getByText('My Example Config').click();
  await expect(sidebar.getByText('Use Ingress?')).toBeVisible();

  const configArea = page.getByTestId('config-area');
  await expect(configArea).toBeVisible();
  await expect(configArea).toContainText('An example field to toggle inclusion of an Ingress Object');

  await page.getByRole('button', { name: 'Continue' }).click();
};
