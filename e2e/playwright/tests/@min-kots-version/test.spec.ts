import { test, expect, Page } from '@playwright/test';
import { login, uploadLicense, downloadAirgapBundle } from '../shared';
import * as constants from './constants';

test('min kots version', async ({ page }) => {
  test.slow();
  await login(page);
  await uploadLicense(page, expect);
  await validateAirgapInstall(page);
});

const validateAirgapInstall = async (page: Page) => {
  await expect(page.locator("#app")).toContainText("Install in airgapped environment", { timeout: 15000 });
  await page.getByTestId("download-app-from-internet").click();

  await page.getByTestId("airgap-registry-hostname").click();
  await page.getByTestId("airgap-registry-hostname").fill('ttl.sh');
  await page.getByTestId("airgap-registry-username").click();
  await page.getByTestId("airgap-registry-username").fill('admin');
  await page.getByTestId("airgap-registry-password").click();
  await page.getByTestId("airgap-registry-password").fill('password');
  await page.getByTestId("airgap-registry-namespace").click();
  await page.getByTestId("airgap-registry-namespace").fill('test');

  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_RESTRICTIVE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    '/tmp/app.airgap'
  );

  await page.setInputFiles('[data-testid="airgap-bundle-drop-zone"] input', '/tmp/app.airgap');
  await page.getByTestId("upload-airgap-bundle-button").click();

  const errorMessage = await page.getByTestId("airgap-bundle-upload-error");
  await expect(errorMessage).toContainText("requires");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);
};

