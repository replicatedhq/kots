import { Page, Expect } from '@playwright/test';

import { validateClusterAdminPreflightResults } from './preflights';

export const validateCurrentVersionCard = async (page: Page, expect: Expect, versionLabel: string, sequence: number) => {
  const currentVersionCard = page.getByTestId("current-version-card");
  await expect(currentVersionCard).toBeVisible();
  await expect(currentVersionCard).toContainText(versionLabel);
  await expect(currentVersionCard).toContainText(`Sequence ${sequence}`);
};

export const validateCurrentReleaseNotes = async (page: Page, expect: Expect, releaseNotes: string) => {
  const currentVersionCard = page.getByTestId("current-version-card");
  await currentVersionCard.getByTestId("current-release-notes-icon").click();
  const releaseNotesModal = page.getByTestId("release-notes-modal");
  await expect(releaseNotesModal).toBeVisible();
  await expect(releaseNotesModal).toContainText(releaseNotes);
  await releaseNotesModal.getByRole("button", { name: "Close" }).click();
  await expect(releaseNotesModal).not.toBeVisible();
};

export const validateCurrentClusterAdminPreflights = async (page: Page, expect: Expect) => {
  const currentVersionCard = page.getByTestId("current-version-card");
  await currentVersionCard.getByTestId("preflight-icon").click();
  await validateClusterAdminPreflightResults(page, expect, 15000);
};

export const validateVersionHistoryAutomaticUpdates = async (page: Page, expect: Expect) => {
  await page.getByText('Configure automatic updates').click();
  const automaticUpdatesModal = page.getByTestId('automatic-updates-modal');
  await expect(automaticUpdatesModal).toBeVisible();

  await automaticUpdatesModal.locator(".replicated-select__control").click();
  await page.waitForTimeout(1000);
  await automaticUpdatesModal.locator(".replicated-select__option").getByText("Weekly", { exact: true }).click();
  await page.waitForTimeout(1000);
  await expect(automaticUpdatesModal.getByTestId("update-checker-spec")).toHaveValue("@weekly");
  await expect(automaticUpdatesModal).toContainText("At 12:00 AM, only on Sunday");

  await expect(automaticUpdatesModal).toContainText("Enable automatic deployment");

  await automaticUpdatesModal.getByRole("button", { name: "Update", exact: true }).click();
  await expect(automaticUpdatesModal).not.toBeVisible();
};
