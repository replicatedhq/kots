import type { Page, Request } from '@playwright/test';

import { IDENTITY_SERVICE_OKTA_DOMAIN, IDENTITY_SERVICE_OKTA_USERNAME } from './constants';

export const loginWithOkta = async (page: Page) => {
  // Restart from the console to create fresh OIDC state after a network change.
  // Do not retry authentication errors or unrelated resource failures.
  for (let attempt = 0; attempt < 3; attempt++) {
    let networkChanged = false;
    const requestFailed = (request: Request) => {
      if (request.failure()?.errorText !== 'net::ERR_NETWORK_CHANGED') {
        return;
      }
      const isOktaRequest = new URL(request.url()).hostname === IDENTITY_SERVICE_OKTA_DOMAIN &&
        ['document', 'xhr', 'fetch'].includes(request.resourceType());
      const isMainNavigation = request.isNavigationRequest() && request.frame() === page.mainFrame();
      if (isOktaRequest || isMainNavigation) {
        networkChanged = true;
      }
    };
    page.on('requestfailed', requestFailed);
    try {
      if (attempt > 0) {
        await page.goto('/');
      }
      // These attributes come from inspecting the Okta login page.
      await page.getByText('Log in with Okta').click();
      await page.locator('input[name="identifier"]').fill(IDENTITY_SERVICE_OKTA_USERNAME);
      await page.locator('input[type="submit"]').click(); // Next button
      await page.locator('input[name="credentials.passcode"]').fill(process.env.IDENTITY_SERVICE_OKTA_PASSWORD!);
      await page.locator('input[type="submit"]').click(); // Verify button
      await page.getByTestId('navbar-dropdown-container').waitFor({ state: 'visible', timeout: 30000 });
      return;
    } catch (error) {
      if (!networkChanged || attempt === 2) {
        throw error;
      }
      console.log(`Okta login interrupted by ERR_NETWORK_CHANGED; restarting login (retry ${attempt + 1}/2).`);
    } finally {
      page.off('requestfailed', requestFailed);
    }
  }
};
