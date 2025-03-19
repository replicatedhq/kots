import { Page, Expect } from '@playwright/test';
import * as uuid from "uuid";

import { runCommand, waitForVeleroAndNodeAgent } from './cli';
import { login } from './login';
import { validateCurrentlyDeployedVersionInfo } from './version-history';
import { validateDashboardInfo } from './dashboard';

import {
  AWS_BUCKET_NAME,
  AWS_REGION,
  APP_SLUG,
  SNAPSHOTS_HOST_PATH
} from "./constants";

export const addSnapshotsRBAC = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();

  const configureSnapshotsModal = page.getByTestId("configure-snapshots-modal");
  await expect(configureSnapshotsModal).toBeVisible({ timeout: 15000 });

  await expect(configureSnapshotsModal.getByTestId("velero-not-installed-tab")).toBeVisible();
  await expect(configureSnapshotsModal.getByTestId("velero-already-installed-tab")).toBeVisible();
  await expect(configureSnapshotsModal.getByTestId("velero-status-box")).toBeVisible();
  
  await configureSnapshotsModal.getByTestId("velero-already-installed-tab").click();
  await expect(configureSnapshotsModal.getByTestId("velero-status-box")).toBeVisible();
  await expect(configureSnapshotsModal.getByTestId("velero-namespace-access-required")).toBeVisible();

  const ensurePermissionsSnippet = configureSnapshotsModal.getByTestId('ensure-permissions-command');
  await expect(ensurePermissionsSnippet).toBeVisible();
  let ensurePermissionsCommand = await ensurePermissionsSnippet.locator(".react-prism.language-bash").textContent();
  expect(ensurePermissionsCommand).not.toBeNull();

  ensurePermissionsCommand = ensurePermissionsCommand.replace(/<velero-namespace>/g, "velero");
  runCommand(ensurePermissionsCommand);

  await configureSnapshotsModal.getByRole('button', { name: 'Check for Velero' }).click();
  await expect(configureSnapshotsModal.getByTestId("velero-is-installed-message")).toBeVisible({ timeout: 15000 });

  await configureSnapshotsModal.getByRole('button', { name: 'Ok, got it!' }).click();
  await expect(configureSnapshotsModal).not.toBeVisible();
};

export const configureSnapshotsAWSInstanceRole = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Settings & Schedule' }).click();

  const storageSettingsCard = page.getByTestId('snapshots-storage-settings-card');
  await expect(storageSettingsCard).toBeVisible({ timeout: 15000 });

  await storageSettingsCard.locator('.replicated-select__control').click();
  await page.waitForTimeout(1000);
  await storageSettingsCard.locator('.replicated-select__option').getByText('Amazon S3', { exact: true }).click();
  await page.waitForTimeout(1000);
  await expect(storageSettingsCard.getByTestId('storage-destination')).toContainText('Amazon S3');

  await storageSettingsCard.getByTestId('aws-bucket').fill(AWS_BUCKET_NAME);
  await storageSettingsCard.getByTestId('aws-region').fill(AWS_REGION);
  await storageSettingsCard.getByTestId('aws-prefix').fill(uuid.v4());
  await storageSettingsCard.getByTestId('aws-use-instance-role').click();
  await expect(storageSettingsCard.getByTestId('aws-access-key-id')).not.toBeVisible();
  await expect(storageSettingsCard.getByTestId('aws-secret-access-key')).not.toBeVisible();

  await storageSettingsCard.getByTestId('update-storage-settings-button').click();
  await expect(storageSettingsCard.getByTestId('storage-settings-updated-confirmation')).toBeVisible({ timeout: 15000 });
  await expect(storageSettingsCard.getByTestId('storage-settings-updated-confirmation')).not.toBeVisible({ timeout: 15000 });

  // wait for velero to be ready
  await waitForVeleroAndNodeAgent();
};

export const validateSnapshotsAWSConfig = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Settings & Schedule' }).click();

  const storageSettingsCard = page.getByTestId('snapshots-storage-settings-card');
  await expect(storageSettingsCard).toBeVisible({ timeout: 15000 });

  await expect(storageSettingsCard.getByTestId('storage-destination')).toContainText('Amazon S3');
  await expect(storageSettingsCard.getByTestId('aws-bucket')).toHaveValue(AWS_BUCKET_NAME);
  await expect(storageSettingsCard.getByTestId('aws-region')).toHaveValue(AWS_REGION);
};

export const validateSnapshotsHostPathConfig = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Settings & Schedule' }).click();

  const storageSettingsCard = page.getByTestId('snapshots-storage-settings-card');
  await expect(storageSettingsCard).toBeVisible({ timeout: 15000 });

  await expect(storageSettingsCard.getByTestId('storage-destination')).toContainText('Host Path');
  await expect(storageSettingsCard.getByTestId('snapshot-hostpath-input')).toHaveValue(SNAPSHOTS_HOST_PATH);
};

export const validateSnapshotsInternalConfig = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Settings & Schedule' }).click();

  const storageSettingsCard = page.getByTestId('snapshots-storage-settings-card');
  await expect(storageSettingsCard).toBeVisible({ timeout: 15000 });
  await expect(storageSettingsCard.getByTestId('storage-destination')).toContainText('Internal Storage (Default)');
};

export const validateAutomaticFullSnapshots = async (page: Page, expect: Expect) => {
  const snapshotsScheduleCard = page.getByTestId('snapshots-schedule-card');
  await expect(snapshotsScheduleCard).toBeVisible();
  await snapshotsScheduleCard.getByTestId('full-snapshots-schedule-tab').click();

  const enableScheduledSnapshotsCheckbox = snapshotsScheduleCard.getByTestId('enable-scheduled-snapshots-checkbox');
  const enableScheduledSnapshotsLabel = snapshotsScheduleCard.getByTestId('enable-scheduled-snapshots-label');
  if (!await enableScheduledSnapshotsCheckbox.isChecked()) {
    await enableScheduledSnapshotsLabel.click();
    await expect(enableScheduledSnapshotsCheckbox).toBeChecked();
  }

  // Schedule interval
  const snapshotsScheduleInterval = snapshotsScheduleCard.getByTestId('snapshots-schedule-interval');
  await snapshotsScheduleInterval.locator(".replicated-select__control").click();
  await page.waitForTimeout(1000);
  await snapshotsScheduleInterval.locator(".replicated-select__option").getByText("Custom", { exact: true }).click();
  await page.waitForTimeout(1000);
  await expect(snapshotsScheduleInterval).toContainText('Custom');

  // Schedule cron expression
  await snapshotsScheduleCard.getByTestId('snapshots-schedule-cron-expression').fill('0 0 * * SUN');
  await expect(snapshotsScheduleCard.getByTestId('snapshots-schedule-human-readable-cron-expression')).toHaveText('At 12:00 AM, only on Sunday');
  await snapshotsScheduleCard.getByRole('button', { name: 'Update schedule', exact: true }).click();
  await expect(snapshotsScheduleCard.getByTestId('snapshots-schedule-update-confirmation')).toBeVisible({ timeout: 15000 });
  await expect(snapshotsScheduleCard.getByTestId('snapshots-schedule-update-confirmation')).not.toBeVisible({ timeout: 15000 });

  const snapshotsRetentionPolicyCard = page.getByTestId('snapshots-retention-policy-card');
  await expect(snapshotsRetentionPolicyCard).toBeVisible();

  // Retention unit
  const snapshotsRetentionUnit = snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-unit');
  await snapshotsRetentionUnit.locator(".replicated-select__control").click();
  await page.waitForTimeout(1000);
  await snapshotsRetentionUnit.locator(".replicated-select__option").getByText("Weeks", { exact: true }).click();
  await page.waitForTimeout(1000);
  await expect(snapshotsRetentionUnit).toContainText('Weeks');

  // Retention value
  await snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-value').fill('2');
  await snapshotsRetentionPolicyCard.getByRole('button', { name: 'Update retention policy', exact: true }).click();
  await expect(snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-policy-update-confirmation')).toBeVisible({ timeout: 15000 });
  await expect(snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-policy-update-confirmation')).not.toBeVisible({ timeout: 15000 });
};

export const validateAutomaticPartialSnapshots = async (page: Page, expect: Expect) => {
  const snapshotsScheduleCard = page.getByTestId('snapshots-schedule-card');
  await expect(snapshotsScheduleCard).toBeVisible();

  await snapshotsScheduleCard.getByTestId('partial-snapshots-schedule-tab').click();
  await expect(snapshotsScheduleCard.getByTestId('partial-snapshots-schedule-app-select')).toBeVisible();

  const enableScheduledSnapshotsCheckbox = snapshotsScheduleCard.getByTestId('enable-scheduled-snapshots-checkbox');
  const enableScheduledSnapshotsLabel = snapshotsScheduleCard.getByTestId('enable-scheduled-snapshots-label');
  if (!await enableScheduledSnapshotsCheckbox.isChecked()) {
    await enableScheduledSnapshotsLabel.click();
    await expect(enableScheduledSnapshotsCheckbox).toBeChecked();
  }

  // Schedule interval
  const snapshotsScheduleInterval = snapshotsScheduleCard.getByTestId('snapshots-schedule-interval');
  await snapshotsScheduleInterval.locator(".replicated-select__control").click();
  await page.waitForTimeout(1000);
  await snapshotsScheduleInterval.locator(".replicated-select__option").getByText("Custom", { exact: true }).click();
  await page.waitForTimeout(1000);
  await expect(snapshotsScheduleInterval).toContainText('Custom');

  // Schedule cron expression
  await snapshotsScheduleCard.getByTestId('snapshots-schedule-cron-expression').fill('0 0 * * SAT');
  await expect(snapshotsScheduleCard.getByTestId('snapshots-schedule-human-readable-cron-expression')).toHaveText('At 12:00 AM, only on Saturday');
  await snapshotsScheduleCard.getByRole('button', { name: 'Update schedule', exact: true }).click();
  await expect(snapshotsScheduleCard.getByTestId('snapshots-schedule-update-confirmation')).toBeVisible({ timeout: 15000 });
  await expect(snapshotsScheduleCard.getByTestId('snapshots-schedule-update-confirmation')).not.toBeVisible({ timeout: 15000 });

  const snapshotsRetentionPolicyCard = page.getByTestId('snapshots-retention-policy-card');
  await expect(snapshotsRetentionPolicyCard).toBeVisible();

  // Retention unit
  const snapshotsRetentionUnit = snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-unit');
  await snapshotsRetentionUnit.locator(".replicated-select__control").click();
  await page.waitForTimeout(1000);
  await snapshotsRetentionUnit.locator(".replicated-select__option").getByText("Years", { exact: true }).click();
  await page.waitForTimeout(1000);
  await expect(snapshotsRetentionUnit).toContainText('Years');

  // Retention value
  await snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-value').fill('3');
  await snapshotsRetentionPolicyCard.getByRole('button', { name: 'Update retention policy', exact: true }).click();
  await expect(snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-policy-update-confirmation')).toBeVisible({ timeout: 15000 });
  await expect(snapshotsRetentionPolicyCard.getByTestId('snapshots-retention-policy-update-confirmation')).not.toBeVisible({ timeout: 15000 });
};

export const createAppSnapshot = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Partial Snapshots (Application)' }).click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId('partial-snapshots-recommendation')).toBeVisible();

  // Partial snapshots page
  const partialSnapshotsCard = page.getByTestId('partial-snapshots-card');
  await expect(partialSnapshotsCard).toBeVisible();
  await expect(partialSnapshotsCard.getByTestId('partial-snapshots-app-select')).toBeVisible();
  await expect(partialSnapshotsCard.getByTestId('partial-snapshots-settings-link')).toBeVisible();
  
  // Create a snapshot
  await partialSnapshotsCard.getByRole('button', { name: 'Start a snapshot', exact: true }).click();
  const snapshotRow = partialSnapshotsCard.getByTestId('snapshot-row-0');
  await expect(snapshotRow).toBeVisible({ timeout: 15000 });
  await expect(snapshotRow).toContainText('In Progress', { timeout: 15000 });
  await expect(snapshotRow).toContainText('Completed', { timeout: 180000 });

  // Verify snapshot size
  await expect(snapshotRow.getByTestId('snapshot-volume-size')).toBeVisible();
  const volumeSize = await snapshotRow.getByTestId('snapshot-volume-size').innerText();
  if (!process.env.DISABLE_SNAPSHOTS_VOLUME_ASSERTIONS) {
    const volumeSizeInt = parseFloat(volumeSize.replace(/MB/g, ""));
    expect(volumeSizeInt).toBeGreaterThan(0);
  }

  // Snapshot details
  await snapshotRow.click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  const snapshotDetailsModal = page.getByTestId('snapshot-details-modal');
  await expect(snapshotDetailsModal).toBeVisible();
  await expect(snapshotDetailsModal.getByTestId('snapshot-volume-size')).toBeVisible();
  await expect(snapshotDetailsModal.getByTestId('snapshot-volume-size')).toContainText(volumeSize);
  await expect(snapshotDetailsModal.getByTestId('snapshot-status')).toContainText('Completed');

  await snapshotDetailsModal.getByTestId('snapshot-type-legacy').click();
  const legacyContent = snapshotDetailsModal.getByTestId('snapshot-type-legacy-content');
  await expect(legacyContent).toBeVisible();

  // Snapshot logs
  await legacyContent.getByTestId('view-logs-button').click();
  const snapshotLogsModal = page.getByTestId('snapshot-logs-modal');
  await expect(snapshotLogsModal).toBeVisible();
  await expect(snapshotLogsModal).toContainText('level=info');
  await snapshotLogsModal.getByRole('button', { name: 'Ok, got it!', exact: true }).click();
  await expect(snapshotLogsModal).not.toBeVisible();

  // Snapshot timeline
  await expect(legacyContent.getByTestId('snapshot-timeline')).toBeVisible();

  // Snapshot volumes
  const snapshotVolumesCard = legacyContent.getByTestId('snapshot-volumes-card');
  await expect(snapshotVolumesCard).toBeVisible();
  const snapshotVolumeRow = snapshotVolumesCard.getByTestId('snapshot-volume-row-0');
  await expect(snapshotVolumeRow).toBeVisible();
  await expect(snapshotVolumeRow).toContainText('Completed');

  // Pre-snapshot scripts
  const snapshotScriptsCard = legacyContent.getByTestId('snapshot-scripts-card');
  await expect(snapshotScriptsCard).toBeVisible();

  const preSnapshotScriptsTab = snapshotScriptsCard.locator('.tab-item').getByText('Pre-snapshot scripts');
  await expect(preSnapshotScriptsTab).toBeVisible();
  await expect(preSnapshotScriptsTab).toHaveClass(/is-active/);

  const preSnapshotScriptRow = snapshotScriptsCard.getByTestId('pre-snapshot-script-row-0');
  await expect(preSnapshotScriptRow).toBeVisible();
  await expect(preSnapshotScriptRow).toContainText('/bin/uname -a');
  await expect(preSnapshotScriptRow).toContainText('Completed');

  // Pre-snapshot script output
  await preSnapshotScriptRow.getByTestId('view-output').click();
  const preSnapshotScriptOutputModal = page.getByTestId('snapshot-script-output-modal');
  await expect(preSnapshotScriptOutputModal).toBeVisible();
  await preSnapshotScriptOutputModal.getByTestId('stdout-tab').click();
  await expect(preSnapshotScriptOutputModal.getByTestId('script-output-editor')).toContainText('Linux');
  await preSnapshotScriptOutputModal.getByTestId('stderr-tab').click();
  await expect(preSnapshotScriptOutputModal.getByTestId('script-output-editor')).not.toContainText('Linux');
  await preSnapshotScriptOutputModal.getByRole('button', { name: 'Ok, got it!', exact: true }).click();
  await expect(preSnapshotScriptOutputModal).not.toBeVisible();

  // Post-snapshot scripts
  const postSnapshotScriptsTab = snapshotScriptsCard.locator('.tab-item').getByText('Post-snapshot scripts');
  await expect(postSnapshotScriptsTab).toBeVisible();
  await expect(postSnapshotScriptsTab).not.toHaveClass(/is-active/);
  await postSnapshotScriptsTab.click();
  await expect(postSnapshotScriptsTab).toHaveClass(/is-active/);
  await expect(preSnapshotScriptsTab).not.toHaveClass(/is-active/);

  const postSnapshotScriptRow = snapshotScriptsCard.getByTestId('post-snapshot-script-row-0');
  await expect(postSnapshotScriptRow).toBeVisible();
  await expect(postSnapshotScriptRow).toContainText('/bin/uname -a');
  await expect(postSnapshotScriptRow).toContainText('Completed');

  // Post-snapshot script output
  await postSnapshotScriptRow.getByTestId('view-output').click();
  const postSnapshotScriptOutputModal = page.getByTestId('snapshot-script-output-modal');
  await expect(postSnapshotScriptOutputModal).toBeVisible();
  await postSnapshotScriptOutputModal.getByTestId('stdout-tab').click();
  await expect(postSnapshotScriptOutputModal.getByTestId('script-output-editor')).toContainText('Linux');
  await postSnapshotScriptOutputModal.getByTestId('stderr-tab').click();
  await expect(postSnapshotScriptOutputModal.getByTestId('script-output-editor')).not.toContainText('Linux');
  await postSnapshotScriptOutputModal.getByRole('button', { name: 'Ok, got it!', exact: true }).click();
  await expect(postSnapshotScriptOutputModal).not.toBeVisible();

  // Return to previous page
  await page.getByTestId('back-button').click();
};

export const restoreAppSnapshot = async (
  page: Page,
  expect: Expect,
  expectedIndex: number,
  expectedSequence: number,
  expectedUpToDate: boolean,
  isAirgapped: boolean
) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Partial Snapshots (Application)' }).click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  const partialSnapshotsCard = page.getByTestId('partial-snapshots-card');
  await expect(partialSnapshotsCard).toBeVisible();

  const snapshotRow = partialSnapshotsCard.getByTestId('snapshot-row-0');
  await expect(snapshotRow).toBeVisible({ timeout: 15000 });

  await snapshotRow.getByTestId('snapshot-restore-button').click();
  const restoreSnapshotModal = page.getByTestId('restore-snapshot-modal');
  await expect(restoreSnapshotModal).toBeVisible();

  await restoreSnapshotModal.getByTestId('app-slug-input').click();
  await restoreSnapshotModal.getByTestId('app-slug-input').fill(APP_SLUG);
  await restoreSnapshotModal.getByRole('button', { name: 'Confirm and restore', exact: true }).click();

  await expect(page.getByTestId('restore-in-progress-view')).toBeVisible({ timeout: 20000 });
  await expect(page.getByTestId('restore-in-progress-title')).toBeVisible();
  await expect(page.getByTestId('restore-in-progress-description')).toBeVisible();
  await expect(page.getByTestId('restore-in-progress-loader')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Cancel snapshot restore', exact: true })).toBeVisible();

  if (!process.env.DISABLE_SNAPSHOTS_VOLUME_ASSERTIONS) {
    const restoreInProgressVolume = page.getByTestId('restore-in-progress-volume-0');
    await expect(restoreInProgressVolume).toBeVisible({ timeout: 210000 });
  }

  await expect(page.getByTestId('restore-completed-view')).toBeVisible({ timeout: 300000 });
  await page.getByRole('link', { name: 'Log in to dashboard', exact: true }).click();
  await expect(page.getByTestId('login-password-input')).toBeVisible({ timeout: 210000 });

  await login(page);
  await validateCurrentlyDeployedVersionInfo(page, expect, expectedIndex, expectedSequence, expectedUpToDate);
  await validateDashboardInfo(page, expect, isAirgapped);
};

export const deleteAppSnapshot = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Partial Snapshots (Application)' }).click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  const partialSnapshotsCard = page.getByTestId('partial-snapshots-card');
  await expect(partialSnapshotsCard).toBeVisible();

  const snapshotRow = partialSnapshotsCard.getByTestId('snapshot-row-0');
  await expect(snapshotRow).toBeVisible({ timeout: 15000 });

  await snapshotRow.getByTestId('snapshot-delete-button').click();
  const deleteSnapshotModal = page.getByTestId('delete-snapshot-modal');
  await expect(deleteSnapshotModal).toBeVisible();
  await deleteSnapshotModal.getByRole('button', { name: 'Delete snapshot', exact: true }).click();
  await expect(deleteSnapshotModal).not.toBeVisible();

  await expect(snapshotRow).toContainText('Deleting', { timeout: 15000 });
  await expect(snapshotRow).not.toBeVisible({ timeout: 180000 });
};

export const createFullSnapshot = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Full Snapshots (Instance)' }).click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  // Full snapshots page
  const fullSnapshotsCard = page.getByTestId('full-snapshots-card');
  await expect(fullSnapshotsCard).toBeVisible();
  await expect(fullSnapshotsCard.getByTestId('full-snapshots-card-title')).toBeVisible();
  await expect(fullSnapshotsCard.getByTestId('full-snapshots-card-description')).toBeVisible();

  // Create a snapshot
  await fullSnapshotsCard.getByRole('button', { name: 'Start a snapshot', exact: true }).click();
  const snapshotRow = fullSnapshotsCard.getByTestId('snapshot-row-0');
  await expect(snapshotRow).toBeVisible({ timeout: 15000 });
  await expect(snapshotRow).toContainText('In Progress', { timeout: 15000 });
  await expect(snapshotRow).toContainText('Completed', { timeout: 300000 });

  // Verify snapshot size
  await expect(snapshotRow.getByTestId('snapshot-volume-size')).toBeVisible();
  const volumeSize = await snapshotRow.getByTestId('snapshot-volume-size').innerText();
  if (!process.env.DISABLE_SNAPSHOTS_VOLUME_ASSERTIONS) {
    const volumeSizeInt = parseFloat(volumeSize.replace(/MB/g, ""));
    expect(volumeSizeInt).toBeGreaterThan(0);
  }

  // Snapshot details
  await snapshotRow.click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  const snapshotDetailsModal = page.getByTestId('snapshot-details-modal');
  await expect(snapshotDetailsModal).toBeVisible();
  await expect(snapshotDetailsModal.getByTestId('snapshot-volume-size')).toBeVisible();
  await expect(snapshotDetailsModal.getByTestId('snapshot-volume-size')).toContainText(volumeSize);
  await expect(snapshotDetailsModal.getByTestId('snapshot-status')).toContainText('Completed');

  await snapshotDetailsModal.getByTestId('snapshot-type-legacy').click();
  const legacyContent = snapshotDetailsModal.getByTestId('snapshot-type-legacy-content');
  await expect(legacyContent).toBeVisible();

  // Snapshot logs
  await legacyContent.getByTestId('view-logs-button').click();
  const snapshotLogsModal = page.getByTestId('snapshot-logs-modal');
  await expect(snapshotLogsModal).toBeVisible();
  await expect(snapshotLogsModal).toContainText('level=info');
  await snapshotLogsModal.getByRole('button', { name: 'Ok, got it!', exact: true }).click();
  await expect(snapshotLogsModal).not.toBeVisible();

  // Snapshot timeline
  await expect(legacyContent.getByTestId('snapshot-timeline')).toBeVisible();

  // Snapshot volumes
  const snapshotVolumesCard = legacyContent.getByTestId('snapshot-volumes-card');
  await expect(snapshotVolumesCard).toBeVisible();
  const snapshotVolumeRow = snapshotVolumesCard.getByTestId('snapshot-volume-row-0');
  await expect(snapshotVolumeRow).toBeVisible();
  await expect(snapshotVolumeRow).toContainText('Completed');

  await snapshotVolumesCard.getByTestId('show-all-volumes-button').click();
  const showAllModal = page.getByTestId('show-all-modal');
  await expect(showAllModal).toBeVisible();
  await expect(showAllModal.getByTestId('snapshot-volume-row-0')).toBeVisible();
  await expect(showAllModal.getByTestId('snapshot-volume-row-0')).toContainText('Completed');
  await expect(showAllModal.getByTestId('snapshot-volume-row-1')).toBeVisible();
  await expect(showAllModal.getByTestId('snapshot-volume-row-1')).toContainText('Completed');
  await showAllModal.getByRole('button', { name: 'Ok, got it!', exact: true }).click();
  await expect(showAllModal).not.toBeVisible();

  // Pre-snapshot scripts
  const snapshotScriptsCard = legacyContent.getByTestId('snapshot-scripts-card');
  await expect(snapshotScriptsCard).toBeVisible();

  const preSnapshotScriptsTab = snapshotScriptsCard.locator('.tab-item').getByText('Pre-snapshot scripts');
  await expect(preSnapshotScriptsTab).toBeVisible();
  await expect(preSnapshotScriptsTab).toHaveClass(/is-active/);

  await expect(snapshotScriptsCard.getByTestId('pre-snapshot-script-row-0')).toBeVisible();
  await expect(snapshotScriptsCard.getByTestId('pre-snapshot-script-row-0')).toContainText('Completed');
  await expect(snapshotScriptsCard.getByTestId('pre-snapshot-script-row-1')).toBeVisible();
  await expect(snapshotScriptsCard.getByTestId('pre-snapshot-script-row-1')).toContainText('Completed');

  // All pre-snapshot scripts
  if (await snapshotScriptsCard.getByTestId('show-all-pre-scripts-button').isVisible()) {
    await snapshotScriptsCard.getByTestId('show-all-pre-scripts-button').click();
    await expect(showAllModal).toBeVisible();
    await expect(showAllModal).toContainText('/bin/uname -a');
    await expect(showAllModal).toContainText('/backup.sh');
    await expect(showAllModal.getByTestId('pre-snapshot-script-row-0')).toBeVisible();
    await expect(showAllModal.getByTestId('pre-snapshot-script-row-0')).toContainText('Completed');
    await expect(showAllModal.getByTestId('pre-snapshot-script-row-1')).toBeVisible();
    await expect(showAllModal.getByTestId('pre-snapshot-script-row-1')).toContainText('Completed');
    await showAllModal.getByRole('button', { name: 'Ok, got it!', exact: true }).click();
    await expect(showAllModal).not.toBeVisible();
  }

  // Post-snapshot scripts
  const postSnapshotScriptsTab = snapshotScriptsCard.locator('.tab-item').getByText('Post-snapshot scripts');
  await expect(postSnapshotScriptsTab).toBeVisible();
  await expect(postSnapshotScriptsTab).not.toHaveClass(/is-active/);
  await postSnapshotScriptsTab.click();
  await expect(postSnapshotScriptsTab).toHaveClass(/is-active/);
  await expect(preSnapshotScriptsTab).not.toHaveClass(/is-active/);

  const postSnapshotScriptRow = snapshotScriptsCard.getByTestId('post-snapshot-script-row-0');
  await expect(postSnapshotScriptRow).toBeVisible();
  await expect(postSnapshotScriptRow).toContainText('/bin/uname -a');
  await expect(postSnapshotScriptRow).toContainText('Completed');

  // Post-snapshot script output
  await postSnapshotScriptRow.getByTestId('view-output').click();
  const postSnapshotScriptOutputModal = page.getByTestId('snapshot-script-output-modal');
  await expect(postSnapshotScriptOutputModal).toBeVisible();
  await postSnapshotScriptOutputModal.getByTestId('stdout-tab').click();
  await expect(postSnapshotScriptOutputModal.getByTestId('script-output-editor')).toContainText('Linux');
  await postSnapshotScriptOutputModal.getByTestId('stderr-tab').click();
  await expect(postSnapshotScriptOutputModal.getByTestId('script-output-editor')).not.toContainText('Linux');
  await postSnapshotScriptOutputModal.getByRole('button', { name: 'Ok, got it!', exact: true }).click();
  await expect(postSnapshotScriptOutputModal).not.toBeVisible();
};

export const restoreFullSnapshot = async (
  page: Page,
  expect: Expect,
  expectedIndex: number,
  expectedSequence: number,
  expectedUpToDate: boolean,
  isAirgapped: boolean
) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Full Snapshots (Instance)' }).click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  const fullSnapshotsCard = page.getByTestId('full-snapshots-card');
  await expect(fullSnapshotsCard).toBeVisible();

  const snapshotRow = fullSnapshotsCard.getByTestId('snapshot-row-0');
  await expect(snapshotRow).toBeVisible({ timeout: 15000 });

  await snapshotRow.getByTestId('snapshot-restore-button').click();
  const restoreSnapshotModal = page.getByTestId('backup-restore-modal');
  await expect(restoreSnapshotModal).toBeVisible();

  const restoreCommandSnippet = restoreSnapshotModal.getByTestId('restore-command');
  await expect(restoreCommandSnippet).toBeVisible();
  const restoreCommand = await restoreCommandSnippet.locator(".react-prism.language-bash").textContent();
  expect(restoreCommand).not.toBeNull();
  runCommand(restoreCommand);

  await page.waitForTimeout(5000);
  await page.reload();
  await expect(page.getByTestId('root-container')).toBeVisible({ timeout: 15000 });

  if (await page.getByTestId('login-password-input').isVisible()) {
    await login(page);
  }

  await validateCurrentlyDeployedVersionInfo(page, expect, expectedIndex, expectedSequence, expectedUpToDate);
  await validateDashboardInfo(page, expect, isAirgapped);
};

export const deleteFullSnapshot = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Snapshots', { exact: true }).click();
  await page.getByRole('link', { name: 'Full Snapshots (Instance)' }).click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  const fullSnapshotsCard = page.getByTestId('full-snapshots-card');
  await expect(fullSnapshotsCard).toBeVisible();

  const snapshotRow = fullSnapshotsCard.getByTestId('snapshot-row-0');
  await expect(snapshotRow).toBeVisible({ timeout: 15000 });

  await snapshotRow.getByTestId('snapshot-delete-button').click();
  const deleteSnapshotModal = page.getByTestId('delete-snapshot-modal');
  await expect(deleteSnapshotModal).toBeVisible();
  await deleteSnapshotModal.getByRole('button', { name: 'Delete snapshot', exact: true }).click();
  await expect(deleteSnapshotModal).not.toBeVisible();

  await expect(snapshotRow).toContainText('Deleting', { timeout: 15000 });
  await expect(snapshotRow).not.toBeVisible({ timeout: 180000 });
};
