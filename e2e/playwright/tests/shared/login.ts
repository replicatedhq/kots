import { Expect, Page } from '@playwright/test';

export const login = async (page: Page) => {
  await page.goto('/');
  await page.getByPlaceholder('password').click();
  await page.getByPlaceholder('password').fill('password');
  await page.getByRole('button', { name: 'Log in' }).click();
};

export const logout = async (page: Page, expect: Expect) => {
  const navbarDropdownContainer = page.getByTestId("navbar-dropdown-container");
  await expect(navbarDropdownContainer).toBeVisible();
  await navbarDropdownContainer.getByTestId("navbar-dropdown-button").click();
  await navbarDropdownContainer.getByTestId("log-out").click();
  await expect(page.getByTestId("login-password-input")).toBeVisible({ timeout: 15000 });
  await expect(navbarDropdownContainer).not.toBeVisible();
};
