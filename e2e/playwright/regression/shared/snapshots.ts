import { Page, Expect } from '@playwright/test';

import { ensureVeleroPermissions } from './cli';

export const addSnapshotsRBAC = async (page: Page, expect: Expect, namespace: string) => {
  const configureSnapshotsModal = page.getByTestId("configure-snapshots-modal");
  await expect(configureSnapshotsModal).toBeVisible({ timeout: 10000 });
  await configureSnapshotsModal.getByText("I've already installed Velero").click();
  await expect(configureSnapshotsModal.getByTestId("ensure-permissions-command")).toContainText('kubectl kots velero ensure-permissions --namespace default --velero-namespace <velero-namespace>');
  await configureSnapshotsModal.getByRole('button', { name: 'Ok, got it!' }).click();
  await expect(configureSnapshotsModal).not.toBeVisible();

  ensureVeleroPermissions(namespace);
};
