import { Page, Expect } from '@playwright/test';

export const validateClusterAdminInitialPreflights = async (page: Page, expect: Expect) => {
  await expect(page.getByTestId("preflight-progress-heading")).toContainText("Collecting information");
  await expect(page.getByTestId("preflight-progress-bar")).toBeVisible();
  await expect(page.getByTestId("preflight-progress-status")).toContainText("Gathering details");

  await page.getByText('Ignore Preflights').click();
  const skipPreflightsModal = await page.getByTestId("skip-preflights-modal");
  await expect(skipPreflightsModal).toBeVisible();
  await expect(skipPreflightsModal.getByTestId("ignore-preflights-and-deploy")).toBeVisible();
  await skipPreflightsModal.getByTestId("wait-for-preflights-to-finish").click();
  await expect(skipPreflightsModal).not.toBeVisible();

  const resultsWrapper = await page.getByTestId("preflight-results-wrapper");
  await expect(resultsWrapper.getByTestId("preflight-results-heading")).toBeVisible({ timeout: 120000 });
  await expect(resultsWrapper.getByTestId("preflight-results-rerun-button")).toBeVisible();

  await expect(resultsWrapper.getByTestId("preflight-message-title")).toContainText('Must have at least 1 node in the cluster');
  await expect(resultsWrapper.getByTestId("preflight-message-row")).toContainText('This cluster has enough nodes.');

  await expect(resultsWrapper.getByTestId("preflight-message-title")).toContainText('Required Kubernetes Version');
  await expect(resultsWrapper.getByTestId("preflight-message-row")).toContainText('Your cluster meets the recommended and required versions of Kubernetes.');

  await expect(resultsWrapper.getByTestId("preflight-message-title")).toContainText('Said hi!');
  await expect(resultsWrapper.getByTestId("preflight-message-row")).toContainText('Said hi!');

  await resultsWrapper.getByRole('button', { name: 'Deploy' }).click();
};
