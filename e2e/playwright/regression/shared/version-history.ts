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

export const validateCurrentDeployLogs = async (page: Page, expect: Expect) => {
  const currentVersionCard = page.getByTestId("current-version-card");
  await currentVersionCard.getByTestId("current-deploy-logs-icon").click();

  const deployLogsModal = page.getByTestId("deploy-logs-modal");
  await expect(deployLogsModal).toBeVisible();
  await expect(deployLogsModal).toContainText("dryrunStdout");
  await expect(deployLogsModal).toContainText("dryrunStderr");
  await expect(deployLogsModal).toContainText("applyStdout");
  await expect(deployLogsModal).toContainText("applyStderr");
  await expect(deployLogsModal).toContainText("helmStdout");
  await expect(deployLogsModal).toContainText("helmStderr");

  await deployLogsModal.getByTestId("logs-tab-applyStdout").click();
  const editor = deployLogsModal.getByTestId("deploy-logs-modal-editor");
  await expect(editor).toBeVisible();
  await expect(editor).toContainText("created");

  await deployLogsModal.getByRole("button", { name: "Ok, got it!" }).click();
  await expect(deployLogsModal).not.toBeVisible();
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

export const validateVersionHistoryRows = async (page: Page, expect: Expect, isOnline: boolean) => {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible();

  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).toBeVisible();
  await expect(updateRow).toContainText('1.0.0');
  await expect(updateRow).toContainText('Sequence 1');
  await expect(updateRow).toContainText('Config Change');
  await expect(updateRow).toContainText('View diff');
  await expect(updateRow.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();

  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible();

  const firstRow = allVersionsCard.getByTestId("version-history-row-0");
  await expect(firstRow).toBeVisible();
  await expect(firstRow).toContainText('1.0.0');
  await expect(firstRow).toContainText('Sequence 1');
  await expect(firstRow).toContainText('Config Change');
  await expect(firstRow).toContainText('View diff');
  await expect(firstRow.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();

  const secondRow = allVersionsCard.getByTestId("version-history-row-1");
  await expect(secondRow).toBeVisible();
  await expect(secondRow).toContainText('1.0.0');
  await expect(secondRow).toContainText('Sequence 0');
  await expect(secondRow).toContainText(isOnline ? 'Online Install' : 'Airgap Install');
  await expect(secondRow).toContainText('Currently deployed version');
  await expect(secondRow.getByRole('button', { name: 'Redeploy', exact: true })).toBeVisible();
};

export const deployVersion = async (page: Page, expect: Expect, index: number, sequence: number, isMinimalRBAC: boolean) => {
  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible();

  const versionRow = allVersionsCard.getByTestId(`version-history-row-${index}`);
  await expect(versionRow).toBeVisible();
  await expect(versionRow).toContainText(`Sequence ${sequence}`);

  const preflightChecksLoader = versionRow.getByTestId('preflight-checks-loader');
  await expect(preflightChecksLoader).not.toBeVisible({ timeout: 180000 });

  if (isMinimalRBAC) {
    await versionRow.getByRole('button', { name: 'Deploy', exact: true }).click();
    const deployWarningModal = page.getByTestId('deploy-warning-modal');
    await expect(deployWarningModal).toBeVisible();
    await expect(deployWarningModal.getByTestId('deploy-warning-modal-text')).toBeVisible();
    await deployWarningModal.getByRole('button', { name: 'Deploy this version', exact: true }).click();
    await expect(deployWarningModal).not.toBeVisible();
  } else {
    await versionRow.getByRole('button', { name: 'Deploy', exact: true }).click();
    const confirmDeploymentModal = page.getByTestId('confirm-deployment-modal');
    await expect(confirmDeploymentModal).toBeVisible();
    await expect(confirmDeploymentModal.getByTestId('confirm-deployment-modal-text')).toBeVisible();
    await confirmDeploymentModal.getByRole('button', { name: 'Deploy', exact: true }).click();
    await expect(confirmDeploymentModal).not.toBeVisible();
  }

  await expect(versionRow).toContainText('Deploying', { timeout: 10000 });
  await expect(versionRow).toContainText('Currently deployed version', { timeout: 45000 });
  await expect(versionRow.getByRole('button', { name: 'Redeploy', exact: true })).toBeVisible();

  const previousVersionRow = allVersionsCard.getByTestId(`version-history-row-${index + 1}`);
  await expect(previousVersionRow).toContainText(`Sequence ${sequence - 1}`);
  await expect(previousVersionRow).toContainText('Previously deployed');
  await expect(previousVersionRow.getByRole('button', { name: 'Rollback', exact: true })).toBeVisible();

  const currentVersionCard = page.getByTestId("current-version-card");
  await expect(currentVersionCard).toContainText(`Sequence ${sequence}`);

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible();
  await expect(updatesCard).toContainText("Application up to date.");
};

export const rollbackToVersion = async (page: Page, expect: Expect, index: number, sequence: number) => {
  await page.locator('.NavItem').getByText('Application', { exact: true }).click();
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible({ timeout: 15000 });

  const versionRow = allVersionsCard.getByTestId(`version-history-row-${index}`);
  await expect(versionRow).toBeVisible();
  await expect(versionRow).toContainText(`Sequence ${sequence}`);

  const preflightChecksLoader = versionRow.getByTestId('preflight-checks-loader');
  await expect(preflightChecksLoader).not.toBeVisible({ timeout: 180000 });

  await versionRow.getByRole('button', { name: 'Rollback', exact: true }).click();
  const confirmDeploymentModal = page.getByTestId('confirm-deployment-modal');
  await expect(confirmDeploymentModal).toBeVisible();
  await expect(confirmDeploymentModal.getByTestId('confirm-deployment-modal-text')).toBeVisible();
  await confirmDeploymentModal.getByRole('button', { name: 'Yes, redeploy', exact: true }).click();
  await expect(confirmDeploymentModal).not.toBeVisible();

  await expect(versionRow).toContainText('Deploying', { timeout: 10000 });
  await expect(versionRow).toContainText('Currently deployed version', { timeout: 45000 });
  await expect(versionRow.getByRole('button', { name: 'Redeploy', exact: true })).toBeVisible();

  const nextVersionRow = allVersionsCard.getByTestId(`version-history-row-${index - 1}`);
  await expect(nextVersionRow).toContainText(`Sequence ${sequence + 1}`);
  await expect(nextVersionRow).toContainText('Previously deployed');
  await expect(nextVersionRow.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();

  const previousVersionRow = allVersionsCard.getByTestId(`version-history-row-${index + 1}`);
  await expect(previousVersionRow).toContainText(`Sequence ${sequence - 1}`);
  await expect(previousVersionRow).toContainText('Previously deployed');
  await expect(previousVersionRow.getByRole('button', { name: 'Rollback', exact: true })).toBeVisible();

  const currentVersionCard = page.getByTestId("current-version-card");
  await expect(currentVersionCard).toContainText(`Sequence ${sequence}`);

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible();

  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).toBeVisible();
  await expect(updateRow).toContainText(`Sequence ${sequence + 1}`);
  await expect(updateRow.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();
};

export const validateVersionDiff = async (page: Page, expect: Expect, firstSequence: number, secondSequence: number) => {
  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible();

  const firstRow = allVersionsCard.getByTestId('version-history-row-0');
  await expect(firstRow).toBeVisible();
  await expect(firstRow).toContainText(`Sequence ${firstSequence}`);
  await expect(firstRow).toContainText('3 files changed');

  const secondRow = allVersionsCard.getByTestId('version-history-row-1');
  await expect(secondRow).toContainText(`Sequence ${secondSequence}`);
  await expect(secondRow).toBeVisible();

  await page.getByTestId('select-releases-to-diff-button').click();

  await expect(firstRow.getByTestId('diff-checkbox')).toBeVisible();
  await expect(secondRow.getByTestId('diff-checkbox')).toBeVisible();

  await page.getByTestId('cancel-diff-button').click();

  await expect(firstRow.getByTestId('diff-checkbox')).not.toBeVisible();
  await expect(secondRow.getByTestId('diff-checkbox')).not.toBeVisible();

  await page.getByTestId('select-releases-to-diff-button').click();
  await firstRow.click();
  await secondRow.click();

  await expect(firstRow.getByTestId('diff-checkbox')).toBeVisible();
  await expect(firstRow.getByTestId('diff-checkbox')).toHaveClass(/checked/);
  await expect(secondRow.getByTestId('diff-checkbox')).toBeVisible();
  await expect(secondRow.getByTestId('diff-checkbox')).toHaveClass(/checked/);

  await page.getByTestId('diff-releases-button').click();

  const diffOverlay = page.getByTestId('diff-overlay');
  await expect(diffOverlay).toBeVisible({ timeout: 15000 });
  await expect(diffOverlay).toContainText(`Diffing releases ${secondSequence} and ${firstSequence}`);
  await expect(diffOverlay).toContainText('+3 additions -3 subtractions 6 changes');
  await diffOverlay.getByTestId('diff-back-button').click();
  await expect(diffOverlay).not.toBeVisible();
};

export const validateCurrentlyDeployedVersionInfo = async (page: Page, expect: Expect, expectedIndex: number, expectedSequence: number, expectedUpToDate: boolean) => {
  await page.locator('.NavItem').getByText('Application', { exact: true }).click();
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const currentVersionCard = page.getByTestId("current-version-card");
  await expect(currentVersionCard).toBeVisible({ timeout: 15000 });
  await expect(currentVersionCard).toContainText(`Sequence ${expectedSequence}`);

  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible();
  const versionRow = allVersionsCard.getByTestId(`version-history-row-${expectedIndex}`);
  await expect(versionRow).toBeVisible();
  await expect(versionRow).toContainText(`Sequence ${expectedSequence}`);
  await expect(versionRow).toContainText('Currently deployed version');

  if (expectedUpToDate) {
    const updatesCard = page.getByTestId('available-updates-card');
    await expect(updatesCard).toBeVisible();
    await expect(updatesCard).toContainText("Application up to date.");
  }
};
