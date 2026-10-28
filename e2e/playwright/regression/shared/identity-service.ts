import { Page, Expect } from '@playwright/test';

import { runCommand, waitForDex } from './cli';
import { updateIdentityServiceOktaApp } from './api';
import { validateDashboardInfo } from './dashboard';
import { loginWithOkta } from './okta-login';
import {
  IDENTITY_SERVICE_OKTA_DOMAIN,
  IDENTITY_SERVICE_OKTA_CLIENT_ID,
  IDENTITY_SERVICE_OKTA_APP_ID
} from './constants';

export const validateIdentityService = async (page: Page, expect: Expect, namespace: string, isAirgapped: boolean) => {
  // update the okta app to use the correct redirect uri
  await updateIdentityServiceOktaApp(IDENTITY_SERVICE_OKTA_DOMAIN, IDENTITY_SERVICE_OKTA_APP_ID);

  await page.locator('.NavItem').getByText('Access', { exact: true }).click();
  const identityProviderForm = page.getByTestId('identity-provider-form');
  await expect(identityProviderForm).toBeVisible({ timeout: 15000 });

  const openidRadio = identityProviderForm.getByTestId('openid-radio');
  await expect(openidRadio).toBeVisible();
  await openidRadio.click();

  await identityProviderForm.getByTestId('connector-name-input').fill('Okta');
  await identityProviderForm.getByTestId('issuer-input').fill(`https://${IDENTITY_SERVICE_OKTA_DOMAIN}/oauth2/default`);
  await identityProviderForm.getByTestId('client-id-input').fill(IDENTITY_SERVICE_OKTA_CLIENT_ID);
  await identityProviderForm.getByTestId('client-secret-input').fill(process.env.IDENTITY_SERVICE_OKTA_CLIENT_SECRET!);

  await identityProviderForm.getByTestId('advanced-options-toggle').click();
  const advancedOptionsForm = identityProviderForm.getByTestId('advanced-options-form');
  await expect(advancedOptionsForm).toBeVisible();
  await advancedOptionsForm.getByTestId('user-name-key-input').fill('sub');

  await page.getByTestId('save-provider-settings-button').click();
  await expect(page.getByTestId('provider-settings-saved-confirmation')).toBeVisible();
  await expect(page.getByTestId('provider-settings-saved-confirmation')).not.toBeVisible({ timeout: 15000 });
  await waitForDex(namespace);

  const navbarDropdownContainer = page.getByTestId("navbar-dropdown-container");
  await expect(navbarDropdownContainer).toBeVisible();
  await navbarDropdownContainer.getByTestId("navbar-dropdown-button").click();
  await navbarDropdownContainer.getByTestId("log-out").click();
  await expect(navbarDropdownContainer).not.toBeVisible({ timeout: 30000 });

  await loginWithOkta(page);

  await validateDashboardInfo(page, expect, isAirgapped);

  // re-enable shared password
  runCommand(`kubectl kots identity-service enable-shared-password -n ${namespace}`);
};
