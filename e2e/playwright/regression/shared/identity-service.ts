import { Page, Expect } from '@playwright/test';

import { runCommand, waitForDex } from './cli';
import { validateDashboardInfo } from './dashboard';
import {
  IDENTITY_SERVICE_OKTA_DOMAIN,
  IDENTITY_SERVICE_OKTA_CLIENT_ID,
  IDENTITY_SERVICE_OKTA_USERNAME
} from './constants';

export const validateIdentityService = async (page: Page, expect: Expect, namespace: string, isAirgapped: boolean) => {
  await page.locator('.NavItem').getByText('Access', { exact: true }).click();

  const identityProviderForm = page.getByTestId('identity-provider-form');
  await expect(identityProviderForm).toBeVisible({ timeout: 15000 });

  const openidRadio = identityProviderForm.getByTestId('openid-radio');
  await expect(openidRadio).toBeVisible();
  await openidRadio.click();

  await identityProviderForm.getByTestId('connector-name-input').fill('Okta');
  await identityProviderForm.getByTestId('issuer-input').fill('https://' + IDENTITY_SERVICE_OKTA_DOMAIN);
  await identityProviderForm.getByTestId('client-id-input').fill(IDENTITY_SERVICE_OKTA_CLIENT_ID);
  await identityProviderForm.getByTestId('client-secret-input').fill(process.env.IDENTITY_SERVICE_OKTA_CLIENT_SECRET!);

  await identityProviderForm.getByTestId('advanced-options-toggle').click();
  const advancedOptionsForm = identityProviderForm.getByTestId('advanced-options-form');
  await expect(advancedOptionsForm).toBeVisible();
  await advancedOptionsForm.getByTestId('user-name-key-input').fill('sub');

  await page.getByTestId('save-provider-settings-button').click();
  await expect(page.getByTestId('provider-settings-saved-confirmation')).toBeVisible();
  await waitForDex(namespace);

  const navbarDropdownContainer = page.getByTestId("navbar-dropdown-container");
  await expect(navbarDropdownContainer).toBeVisible();
  await navbarDropdownContainer.getByTestId("navbar-dropdown-button").click();
  await navbarDropdownContainer.getByTestId("log-out").click();
  await expect(navbarDropdownContainer).not.toBeVisible({ timeout: 30000 });

  // these attributes come from inspecting the okta login page
  await page.getByText('Log in with Okta').click();
  await page.locator('input[name="identifier"]').click();
  await page.locator('input[name="identifier"]').fill(IDENTITY_SERVICE_OKTA_USERNAME);
  await page.locator('input[name="credentials.passcode"]').click();
  await page.locator('input[name="credentials.passcode"]').fill(process.env.IDENTITY_SERVICE_OKTA_PASSWORD!);
  await page.locator('input[type="submit"]').click();

  await validateDashboardInfo(page, expect, isAirgapped);

  // re-enable shared password
  runCommand(`kubectl kots identity-service enable-shared-password -n ${namespace}`);
};
