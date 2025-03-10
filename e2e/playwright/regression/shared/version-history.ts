import { Page, Expect } from '@playwright/test';
import * as uuid from "uuid";

import { RegistryInfo, cliAirgapUpdate } from './cli';
import { promoteRelease } from './api';
import {
  validateClusterAdminPreflightResults,
  validateMinimalRBACPreflightsPage
} from './preflights';

export const validateCurrentVersionCard = async (page: Page, expect: Expect, sequence: number) => {
  const currentVersionCard = page.getByTestId("current-version-card");
  await expect(currentVersionCard).toBeVisible();
  await expect(currentVersionCard).toContainText(`Sequence ${sequence}`);
};

export const validateCurrentClusterAdminPreflights = async (page: Page, expect: Expect) => {
  const currentVersionCard = page.getByTestId("current-version-card");
  await currentVersionCard.getByTestId("preflight-icon").click();
  await validateClusterAdminPreflightResults(page, expect, 15000);
  await page.getByTestId("preflight-results-back-button").click();
};

export const validateVersionMinimalRBACPreflights = async (page: Page, expect: Expect, rowIndex: number, sequence: number) => {
  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible({ timeout: 15000 });

  const versionRow = allVersionsCard.getByTestId(`version-history-row-${rowIndex}`);
  await expect(versionRow).toBeVisible();
  await expect(versionRow).toContainText(`Sequence ${sequence}`);

  await versionRow.getByTestId("preflight-icon").click();
  await validateMinimalRBACPreflightsPage(page, expect, 15000);
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
  await expect(editor).toContainText(/created|configured|unchanged/);

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

export const validateVersionHistoryRows = async (page: Page, expect: Expect, isAirgapped: boolean) => {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible();

  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).toBeVisible();
  await expect(updateRow).toContainText('Sequence 2');
  await expect(updateRow).toContainText('Config Change');
  await expect(updateRow).toContainText('View diff');
  await expect(updateRow.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();

  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible();

  const firstRow = allVersionsCard.getByTestId("version-history-row-0");
  await expect(firstRow).toBeVisible();
  await expect(firstRow).toContainText('Sequence 2');
  await expect(firstRow).toContainText('Config Change');
  await expect(firstRow).toContainText('View diff');
  await expect(firstRow.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();

  const secondRow = allVersionsCard.getByTestId("version-history-row-1");
  await expect(secondRow).toBeVisible();
  await expect(secondRow).toContainText('Sequence 1');
  await expect(secondRow).toContainText(isAirgapped ? 'Airgap Update' : 'Upstream Update');
  await expect(secondRow).toContainText('Currently deployed version');
  await expect(secondRow.getByRole('button', { name: 'Redeploy', exact: true })).toBeVisible();

  const thirdRow = allVersionsCard.getByTestId("version-history-row-2");
  await expect(thirdRow).toBeVisible();
  await expect(thirdRow).toContainText('Sequence 0');
  await expect(thirdRow).toContainText(isAirgapped ? 'Airgap Install' : 'Online Install');
  await expect(thirdRow).toContainText('Previously deployed');
  await expect(thirdRow.getByRole('button', { name: 'Rollback', exact: true })).toBeVisible();
};

export const deployNewVersion = async (
  page: Page,
  expect: Expect,
  expectedSequence: number,
  expectedSource: string,
  isMinimalRBAC: boolean,
  skipNavigation: boolean = false,
  supportsRollback: boolean = true
) => {
  if (!skipNavigation) {
    await page.locator('.NavItem').getByText('Application', { exact: true }).click();
    await page.getByRole('link', { name: 'Version history', exact: true }).click();
  }

  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible({ timeout: 15000 });

  const versionRow = allVersionsCard.getByTestId('version-history-row-0');
  await expect(versionRow).toBeVisible();
  await expect(versionRow).toContainText(expectedSource);
  await expect(versionRow).toContainText(`Sequence ${expectedSequence}`);

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
    await confirmDeploymentModal.getByRole('button', { name: 'Yes, deploy', exact: true }).click();
    await expect(confirmDeploymentModal).not.toBeVisible();
  }

  await expect(versionRow).toContainText('Deploying', { timeout: 10000 });
  await expect(versionRow).toContainText('Currently deployed version', { timeout: 45000 });
  await expect(versionRow.getByRole('button', { name: 'Redeploy', exact: true })).toBeVisible();

  const previousVersionRow = allVersionsCard.getByTestId('version-history-row-1');
  await expect(previousVersionRow).toContainText('Previously deployed');
  await expect(previousVersionRow.getByRole('button', { name: 'Rollback', exact: true })).toBeVisible({ visible: supportsRollback });
  await expect(previousVersionRow).toContainText(`Sequence ${expectedSequence - 1}`);

  const currentVersionCard = page.getByTestId("current-version-card");
  await expect(currentVersionCard).toContainText(`Sequence ${expectedSequence}`);

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible();
  await expect(updatesCard).toContainText("Application up to date.");
};

export const rollbackToVersion = async (page: Page, expect: Expect, rowIndex: number, sequence: number) => {
  await page.locator('.NavItem').getByText('Application', { exact: true }).click();
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible({ timeout: 15000 });

  const versionRow = allVersionsCard.getByTestId(`version-history-row-${rowIndex}`);
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

  const nextVersionRow = allVersionsCard.getByTestId(`version-history-row-${rowIndex - 1}`);
  await expect(nextVersionRow).toContainText(`Sequence ${sequence + 1}`);
  await expect(nextVersionRow).toContainText('Previously deployed');
  await expect(nextVersionRow.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();

  const previousVersionRow = allVersionsCard.getByTestId(`version-history-row-${rowIndex + 1}`);
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

export const validateCurrentlyDeployedVersionInfo = async (page: Page, expect: Expect, expectedRowIndex: number, expectedSequence: number, expectedUpToDate: boolean) => {
  await page.locator('.NavItem').getByText('Application', { exact: true }).click();
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const currentVersionCard = page.getByTestId("current-version-card");
  await expect(currentVersionCard).toBeVisible({ timeout: 15000 });
  await expect(currentVersionCard).toContainText(`Sequence ${expectedSequence}`);

  const allVersionsCard = page.getByTestId('all-versions-card');
  await expect(allVersionsCard).toBeVisible();
  const versionRow = allVersionsCard.getByTestId(`version-history-row-${expectedRowIndex}`);
  await expect(versionRow).toBeVisible();
  await expect(versionRow).toContainText(`Sequence ${expectedSequence}`);
  await expect(versionRow).toContainText('Currently deployed version');

  if (expectedUpToDate) {
    const updatesCard = page.getByTestId('available-updates-card');
    await expect(updatesCard).toBeVisible();
    await expect(updatesCard).toContainText("Application up to date.");
  }
};

export const validateCheckForUpdates = async (page: Page, expect: Expect, channelId: string, vendorReleaseSequence: number, expectedSequence: number, isMinimalRBAC: boolean) => {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const newVersionLabel = `1.0.0+${uuid.v4()}`;
  const newReleaseNotes = `notes-${uuid.v4()}`;
  await promoteRelease(vendorReleaseSequence, channelId, newVersionLabel, newReleaseNotes);

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible();

  await updatesCard.getByTestId('check-for-update-button').click();
  await expect(updatesCard.getByTestId('check-for-update-progress').locator('.Loader')).toBeVisible();
  await expect(updatesCard.getByTestId('check-for-update-progress')).toContainText('ing', { timeout: 30000 });
  await expect(updatesCard.getByTestId('check-for-update-progress')).not.toBeVisible({ timeout: 240000 });

  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).toBeVisible();
  await expect(updateRow).toContainText('Upstream Update');
  await expect(updateRow).toContainText(newVersionLabel);

  await updateRow.getByTestId('release-notes-icon').click();
  await validateReleaseNotesModal(page, expect, newReleaseNotes);

  await deployNewVersion(page, expect, expectedSequence, 'Upstream Update', isMinimalRBAC, true);

  const currentVersionCard = page.getByTestId("current-version-card");
  await currentVersionCard.getByTestId("current-release-notes-icon").click();
  await validateReleaseNotesModal(page, expect, newReleaseNotes);
};

export const validateUiAirgapUpdate = async (page: Page, expect: Expect, airgapBundlePath: string) => {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible({ timeout: 15000 });
  await expect(updatesCard).toContainText("Application up to date.");

  await validateCurrentVersionCard(page, expect, 0);

  await page.setInputFiles('[data-testid="airgap-bundle-drop-zone"] input', airgapBundlePath);
  const airgapUploadProgress = page.getByTestId("airgap-upload-progress");
  await expect(airgapUploadProgress).toBeVisible({ timeout: 15000 });
  await expect(airgapUploadProgress.getByTestId("airgap-upload-progress-title")).toBeVisible();
  await expect(airgapUploadProgress.getByTestId("airgap-upload-progress-bar")).toBeVisible();

  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).toBeVisible({ timeout: 45000 });
  await expect(updateRow).toContainText('Airgap Update');

  const preflightChecksLoader = updateRow.getByTestId('preflight-checks-loader');
  await expect(preflightChecksLoader).toBeVisible();
  await expect(preflightChecksLoader).not.toBeVisible({ timeout: 120000 });

  await updateRow.getByTestId('release-notes-icon').click();
  await validateReleaseNotesModal(page, expect, "release notes - updates");

  // minimal rbac is false because we uploaded the initial bundle via the ui.
  // in airgap, minimal rbac is only detected if the bundle is passed to cli install.
  // also, the releases associated with ui installs do not support rollback.
  await deployNewVersion(page, expect, 1, 'Airgap Update', false, false, false);

  const currentVersionCard = page.getByTestId("current-version-card");
  await currentVersionCard.getByTestId("current-release-notes-icon").click();
  await validateReleaseNotesModal(page, expect, "release notes - updates");
};

export const validateCliAirgapUpdate = async (
  page: Page,
  expect: Expect,
  expectedSequence: number,
  isMinimalRBAC: boolean,
  airgapBundlePath: string,
  namespace: string,
  isExistingCluster: boolean,
  registryInfo?: RegistryInfo
) => {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();

  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible({ timeout: 15000 });

  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).not.toBeVisible();

  cliAirgapUpdate(
    airgapBundlePath,
    namespace,
    isExistingCluster,
    registryInfo
  );

  await page.reload();

  await expect(updateRow).toBeVisible({ timeout: 15000 });
  await expect(updateRow).toContainText('Airgap Update');
  await expect(updateRow).toContainText(`Sequence ${expectedSequence}`);

  await updateRow.getByTestId('release-notes-icon').click();
  await validateReleaseNotesModal(page, expect, 'release notes - updates');

  await deployNewVersion(page, expect, expectedSequence, 'Airgap Update', isMinimalRBAC, true);

  const currentVersionCard = page.getByTestId("current-version-card");
  await currentVersionCard.getByTestId("current-release-notes-icon").click();
  await validateReleaseNotesModal(page, expect, "release notes - updates");
}

export const validateReleaseNotesModal = async (page: Page, expect: Expect, releaseNotes: string) => {
  const releaseNotesModal = page.getByTestId("release-notes-modal");
  await expect(releaseNotesModal).toBeVisible();
  await expect(releaseNotesModal).toContainText(releaseNotes);
  await releaseNotesModal.getByRole("button", { name: "Close" }).click();
  await expect(releaseNotesModal).not.toBeVisible();
};
