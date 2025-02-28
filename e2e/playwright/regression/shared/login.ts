import { Page } from '@playwright/test';

export const login = async (page: Page) => {
  await page.getByPlaceholder('password').click();
  await page.getByPlaceholder('password').fill('password');
  await page.getByRole('button', { name: 'Log in' }).click();
};
