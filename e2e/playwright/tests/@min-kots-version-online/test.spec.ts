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

test('min kots version', async ({ page }) => {
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
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);
};

const validateOnlineInstallPermissive = async (page: Page, expect: Expect) => {
  await promoteReleaseBySemver(constants.VENDOR_PERMISSIVE_RELEASE_SEMVER, constants.VENDOR_APP_ID, constants.CHANNEL_ID);

  await airgapOnlineInstall(page, expect);

  await appIsReady(page, expect);
};

const validateOnlineUpdateRestrictive = async (page: Page, expect: Expect) => {
  await promoteReleaseBySemver(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER, constants.VENDOR_APP_ID, constants.CHANNEL_ID);

  await onlineCheckForUpdates(page, expect);

  await page.getByTestId("console-subnav").getByRole("link", { name: "Version history" }).click();

  const availableUpdateCard = page.getByTestId("available-updates-card");
  let card = availableUpdateCard.getByTestId("version-history-row-0");
  await expect(card.getByTestId("version-label")).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);
  await expect(card.getByTestId("version-action-button")).toContainText("Download");
  await expect(card.getByTestId("version-status")).toContainText("Pending download");

  let errorMessage = card.getByTestId("version-download-status");
  await expect(errorMessage).toContainText("requires", { timeout: 5 * 1000 }); // 5 seconds
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);

  var allVersionsCard = page.getByTestId("all-versions-card");
  card = allVersionsCard.getByTestId("version-history-row-0");
  await expect(card.getByTestId("version-label")).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);

  // Click the download button and validate that you cannot download it
  await card.getByTestId("version-action-button").click();
  await page.waitForTimeout(1 * 1000); // 1 second
  await expect(card.getByTestId("version-downloading-status")).not.toContainText("Downloading");

  errorMessage = card.getByTestId("version-downloading-status");
  await expect(errorMessage).toContainText("requires", { timeout: 5 * 1000 }); // 5 seconds
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_MIN_KOTS_VERSION);

  // Click the diff button and validate that you cannot select this version to diff because it was
  // unable to download
  await page.getByTestId("select-releases-to-diff-button").click();
  await expect(card).not.toBeVisible();

  // Click the cancel button and validate that you can see the version card again
  await page.getByTestId("cancel-diff-button").click();
  await expect(card).toBeVisible();

  // A license sync may happen on installation and there will be a "License changed" version in the
  // list
  const versionSequence = card.getByTestId("version-sequence");
  const versionSequenceText = await versionSequence.textContent();
  const versionSequenceNumber = versionSequenceText.match(/\d+/);
  if (!versionSequenceNumber || versionSequenceNumber.length !== 1) {
    throw new Error(`version sequence number not found in text "${versionSequenceText}"`);
  }
  const versionSequenceNumberInt = parseInt(versionSequenceNumber[0]);
  if (isNaN(versionSequenceNumberInt)) {
    throw new Error(`version sequence number is not a number "${versionSequenceNumber}"`);
  }

  // Click the "View files" tab and validate that the url has the correct sequence as the
  // restrictive version was unable to download
  await page.getByTestId("console-subnav").getByRole("link", { name: "View files" }).click();
  await expect(page).toHaveURL(new RegExp(`.*\/tree\/${versionSequenceNumberInt-1}$`));
};
