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

test('range kots version', async ({ page }) => {
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

  await page.getByTestId("console-subnav").getByRole("link", { name: "Version history" }).click();

  const availableUpdateCard = page.getByTestId("available-updates-card");
  const card = availableUpdateCard.getByTestId("version-history-row-0");
  await expect(card.getByTestId("version-label")).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);
  await expect(card.getByTestId("version-action-button")).toContainText("Download");
  await expect(card.getByTestId("version-status")).toContainText("Pending download");

  let errorMessage = card.getByTestId("version-download-status");
  await expect(errorMessage).toContainText("requires", { timeout: 5 * 1000 }); // 5 seconds
  await expect(errorMessage).toContainText("Upgrade KOTS");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_TARGET_KOTS_VERSION);

  const allVersionsCard = page.getByTestId("all-versions-card");
  const versionRow = allVersionsCard.getByTestId("version-history-row-0");
  await expect(versionRow.getByTestId("version-label")).toContainText(constants.VENDOR_RESTRICTIVE_RELEASE_SEMVER);

  // Click the download button and validate that you cannot download it
  await versionRow.getByTestId("version-action-button").click();
  await page.waitForTimeout(1 * 1000); // 1 second
  await expect(versionRow.getByTestId("version-downloading-status")).not.toContainText("Downloading");

  errorMessage = versionRow.getByTestId("version-downloading-status");
  await expect(errorMessage).toContainText("requires", { timeout: 5 * 1000 }); // 5 seconds
  await expect(errorMessage).toContainText("Upgrade KOTS");
  await expect(errorMessage).toContainText(constants.RESTRICTIVE_TARGET_KOTS_VERSION);

  // Click the diff button and validate that you cannot select this version to diff because it was
  // unable to download
  await page.getByTestId("select-releases-to-diff-button").click();
  await expect(versionRow).not.toBeVisible();

  // Click the cancel button and validate that you can see the version card again
  await page.getByTestId("cancel-diff-button").click();
  await expect(versionRow).toBeVisible();

  // A license sync may happen on installation and there will be a "License changed" version in the
  // list
  const versionSequence = versionRow.getByTestId("version-sequence");
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

const validateCliInstallFailsEarly = () => {
  let result = "";
  try {
    execSync(`kubectl kots install ${constants.APP_SLUG}/automated --no-port-forward --namespace ${constants.APP_SLUG} --shared-password password`);
  } catch (error: any) {
    result = error.stderr.toString();
  }
  if (!result.includes("requires") || !result.includes(constants.RESTRICTIVE_TARGET_KOTS_VERSION)) {
    throw new Error(`Expected error message to contain "requires" and "${constants.RESTRICTIVE_TARGET_KOTS_VERSION}" but got: ${result}`);
  }
};
