import { test, expect, Page, Expect } from '@playwright/test';
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

  await airgapOnlineInstall(page, expect);

  const errorMessage = airgapInstallErrorMessage(page);
  await expect(errorMessage).toContainText("requires");
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

  await expect(page.getByTestId("footer-target-kots-version")).toContainText(`${constants.PERMISSIVE_TARGET_KOTS_VERSION} available.`);

  await onlineCheckForUpdates(page, expect);

  const availableUpdateCard = page.getByTestId("available-updates-card");
  await expect(availableUpdateCard).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);

  await expect(page.getByTestId("footer-target-kots-version")).not.toContainText(`${constants.PERMISSIVE_TARGET_KOTS_VERSION} available.`);
};
