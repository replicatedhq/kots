import { test, expect, Page, Expect } from '@playwright/test';
import { execSync } from 'child_process';
import * as constants from './constants';
import {
  login,
  uploadLicense,
  appIsReady,
  airgapInstallErrorMessage,
  airgapOnlineInstall,
  onlineCheckForUpdates,
  promoteReleaseBySemver,
} from '../shared';

test('target kots version', async ({ page }) => {
  test.setTimeout(2 * 60 * 1000); // 2 minutes

  await login(page);
  await uploadLicense(page, expect);
  await validateOnlineInstallRestrictive(page, expect);
  await validateOnlineInstallPermissive(page, expect);
  await validateOnlineUpdateRestrictive(page, expect);
});

const validateOnlineInstallRestrictive = async (page: Page, expect: Expect) => {
  await promoteReleaseBySemver(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER, constants.VENDOR_APP_ID, constants.CHANNEL_ID);

  validateCliInstallFailsEarly();
  await airgapOnlineInstall(page, expect);

  const errorMessage = airgapInstallErrorMessage(page);
  await expect(errorMessage).toContainText("requires");
  await expect(errorMessage).toContainText("Install KOTS");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_TARGET_KOTS_VERSION);
};

const validateOnlineInstallPermissive = async (page: Page, expect: Expect) => {
  await promoteReleaseBySemver(constants.VENDOR_PERMISSIVE_RELEASE_SEMVER, constants.VENDOR_APP_ID, constants.CHANNEL_ID);

  await airgapOnlineInstall(page, expect);

  await appIsReady(page, expect);
};

const validateOnlineUpdateRestrictive = async (page: Page, expect: Expect) => {
  await promoteReleaseBySemver(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER, constants.VENDOR_APP_ID, constants.CHANNEL_ID);

  await page.getByTestId("console-subnav").getByRole("link", { name: "Version history" }).click();

  const footer = page.getByTestId("footer");
  await expect(footer).toContainText(`${constants.PERMISSIVE_TARGET_KOTS_VERSION} available.`);

  await onlineCheckForUpdates(page, expect);

  const availableUpdateCard = page.getByTestId("available-updates-card");
  await expect(availableUpdateCard).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER, { timeout: 30 * 1000 }); // 30 seconds

  await expect(footer).not.toContainText(`${constants.PERMISSIVE_TARGET_KOTS_VERSION} available.`);
};

const validateCliInstallFailsEarly = () => {
  let result = "";
  try {
    execSync(`kubectl kots install ${constants.APP_SLUG}/automated --no-port-forward --namespace ${constants.APP_SLUG} --shared-password password`);
  } catch (error: any) {
    result = error.stderr?.toString();
  }
  if (!result.includes("requires") || !result.includes(constants.RESTRICTIVE_TARGET_KOTS_VERSION)) {
    throw new Error(`Expected error message to contain "requires" and "${constants.RESTRICTIVE_TARGET_KOTS_VERSION}" but got: ${result}`);
  }
};
