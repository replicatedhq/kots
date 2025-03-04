import { test, expect, Page, Expect } from '@playwright/test';
import * as constants from './constants';
import {
  login,
  uploadLicense,
  listReleases,
  promoteRelease,
  appIsReady,
  airgapInstallErrorMessage,
  airgapOnlineInstall,
  onlineCheckForUpdates,
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
  await promoteReleaseBySemver(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);

  await airgapOnlineInstall(page, expect);

  const errorMessage = airgapInstallErrorMessage(page);
  await expect(errorMessage).toContainText("requires");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_TARGET_KOTS_VERSION);
};

const validateOnlineInstallPermissive = async (page: Page, expect: Expect) => {
  await promoteReleaseBySemver(constants.VENDOR_PERMISSIVE_RELEASE_SEMVER);

  await airgapOnlineInstall(page, expect);

  await appIsReady(page, expect);
};

const validateOnlineUpdateRestrictive = async (page: Page, expect: Expect) => {
  await promoteReleaseBySemver(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);

  await onlineCheckForUpdates(page, expect);

  await page.getByTestId("console-subnav").getByRole("link", { name: "Version history" }).click();

  const availableUpdateCard = page.getByTestId("available-updates-card");
  await expect(availableUpdateCard).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);

  const footerTargetKotsVersion = page.getByTestId("footer-target-kots-version");
  await expect(footerTargetKotsVersion).toContainText(`${constants.PERMISSIVE_TARGET_KOTS_VERSION} available.`);
};

const promoteReleaseBySemver = async (semver: string) => {
  const releases = await listReleases(constants.VENDOR_APP_ID, constants.CHANNEL_ID);

  let releaseToPromote = null;
  for (const release of releases) {
    if (release.semver === semver) {
      releaseToPromote = release;
      break;
    }
  }

  if (!releaseToPromote) {
    throw new Error(`release not found for semver ${semver}`);
  }

  await promoteRelease(constants.VENDOR_APP_ID, releaseToPromote.sequence, constants.CHANNEL_ID, semver);
};
