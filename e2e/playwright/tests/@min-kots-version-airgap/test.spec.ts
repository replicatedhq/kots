import { test, expect, Page, Expect } from '@playwright/test';
import * as constants from './constants';
import {
  login,
  uploadLicense,
  downloadAirgapBundle,
  airgapInstall,
  airgapInstallErrorMessage,
  appIsReady,
  airgapUpdate,
  airgapUpdateErrorMessage,
} from '../shared';

test('min kots version', async ({ page }) => {
  test.setTimeout(5 * 60 * 1000); // 5 minutes

  await login(page);
  await uploadLicense(page, expect);
  await validateAirgapInstallRestrictive(page, expect);
  await validateAirgapInstallPermissive(page, expect);
  await validateAirgapUpdateRestrictive(page, expect);
});

const validateAirgapInstallRestrictive = async (page: Page, expect: Expect) => {
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_RESTRICTIVE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    '/tmp/app.airgap'
  );

  await airgapInstall(page, expect, 'ttl.sh', 'admin', 'password', 'test', '/tmp/app.airgap', 15 * 1000); // 15 seconds (should fail quickly)

  const errorMessage = airgapInstallErrorMessage(page);
  await expect(errorMessage).toContainText("requires");
  await expect(errorMessage).toContainText("Install KOTS");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);
};

const validateAirgapInstallPermissive = async (page: Page, expect: Expect) => {
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_PERMISSIVE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    '/tmp/app.airgap'
  );

  await airgapInstall(page, expect, 'ttl.sh', 'admin', 'password', 'test', '/tmp/app.airgap');

  await appIsReady(page, expect);
};

const validateAirgapUpdateRestrictive = async (page: Page, expect: Expect) => {
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.VENDOR_RESTRICTIVE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    '/tmp/app.airgap'
  );

  await airgapUpdate(page, expect, '/tmp/app.airgap');

  const errorMessage = airgapUpdateErrorMessage(page);
  await expect(errorMessage).toContainText("requires");
  await expect(errorMessage).toContainText("Upgrade KOTS");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);
};
