import { Page, Expect } from '@playwright/test';

export const validateClusterAdminInitialPreflights = async (page: Page, expect: Expect) => {
  await expect(page.getByTestId("preflight-progress-heading")).toContainText("Collecting information");
  await expect(page.getByTestId("preflight-progress-bar")).toBeVisible();
  await expect(page.getByTestId("preflight-progress-status")).toContainText("Gathering details");

  await page.getByText('Ignore Preflights').click();
  const skipPreflightsModal = page.getByTestId("skip-preflights-modal");
  await expect(skipPreflightsModal).toBeVisible();
  await expect(skipPreflightsModal.getByTestId("ignore-preflights-and-deploy")).toBeVisible();
  await skipPreflightsModal.getByTestId("wait-for-preflights-to-finish").click();
  await expect(skipPreflightsModal).not.toBeVisible();

  const resultsWrapper = page.getByTestId("preflight-results-wrapper");
  await expect(resultsWrapper.getByTestId("preflight-results-heading")).toBeVisible({ timeout: 120000 });
  await expect(resultsWrapper.getByTestId("preflight-results-rerun-button")).toBeVisible();

  await expect(resultsWrapper.getByTestId("preflight-message-title").first()).toContainText('Must have at least 1 node in the cluster');
  await expect(resultsWrapper.getByTestId("preflight-message-row").first()).toContainText('This cluster has enough nodes.');

  await expect(resultsWrapper.getByTestId("preflight-message-title").nth(1)).toContainText('Required Kubernetes Version');
  await expect(resultsWrapper.getByTestId("preflight-message-row").nth(1)).toContainText('Your cluster meets the recommended and required versions of Kubernetes.');

  await expect(resultsWrapper.getByTestId("preflight-message-title").nth(2)).toContainText('Said hi!');
  // TODO NOW: uncomment this
  // await expect(resultsWrapper.getByTestId("preflight-message-row").nth(2)).toContainText('Said hi!');

  await page.getByRole('button', { name: 'Deploy' }).click();
};
