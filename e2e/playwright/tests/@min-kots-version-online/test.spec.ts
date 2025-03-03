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

test('min kots version', async ({ page }) => {
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
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);
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

  const newVersionCard = page.getByTestId("new-version-card");
  await expect(newVersionCard.getByTestId("version-label")).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);
  await expect(newVersionCard.getByTestId("version-action-button")).toContainText("Download");
  await expect(newVersionCard.getByTestId("version-status")).toContainText("Pending download");

  let errorMessage = newVersionCard.getByTestId("version-download-status");
  await expect(errorMessage).toContainText("requires");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);

  // Click the download button and validate that you once again see the error message
  await newVersionCard.getByTestId("version-action-button").click();
  await page.waitForTimeout(1 * 1000); // 1 second
  await expect(newVersionCard.getByTestId("version-downloading-status")).not.toContainText("Downloading");

  errorMessage = newVersionCard.getByTestId("version-downloading-status");
  await expect(errorMessage).toContainText("requires");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);

  // Click the diff button and validate that you can no longer see the version card
  await newVersionCard.getByTestId("diff-versions-button").click();
  await expect(newVersionCard.getByTestId("version-label")).not.toBeVisible();
  await expect(newVersionCard.getByTestId("version-action-button")).not.toBeVisible();
  await expect(newVersionCard.getByTestId("version-status")).not.toBeVisible();

  // Click the cancel button and validate that you can see the version card again
  await newVersionCard.getByTestId("cancel-diff-button").click();
  await expect(newVersionCard.getByTestId("version-label")).toBeVisible();
  await expect(newVersionCard.getByTestId("version-action-button")).toBeVisible();
  await expect(newVersionCard.getByTestId("version-status")).toBeVisible();

  // Click the "View files" tab and validate that the url has the correct sequence
  await page.getByTestId("console-subnav").getByRole("link", { name: "View files" }).click();
  await expect(page).toHaveURL(/.*\/tree\/0$/);
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
