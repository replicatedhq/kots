import { test, expect, Page } from '@playwright/test';
import { login, uploadLicense } from '../shared';
import { execSync } from 'child_process';
import { NUM_OF_VERSIONS, APP_SLUG, NAMESPACE, DEFAULT_PAGE_SIZE, LATEST_SEQUENCE } from './constants';

test('version history pagination', async ({ page }) => {
  test.setTimeout(300000); // 300 seconds, 5 minutes
  await login(page);
  await uploadLicense(page, expect, "version-history-pagination.yaml");
  await expect(page.locator('#app')).toContainText('Configure Version History Pagination', { timeout: 15000 });
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Currently deployed version', { timeout: 15000 });
  await expect(page.getByTestId("dashboard-app-status")).toContainText("Ready", { timeout: 30000 });
  // create many versions via the CLI
  const startTime = Date.now();
  for (let i = 0; i < NUM_OF_VERSIONS; i++) {
    const commandStartTime = Date.now();
    execSync(`kubectl kots set config ${APP_SLUG} -n ${NAMESPACE} item_1="version-${i}"`);
    const commandEndTime = Date.now();
    const commandDuration = (commandEndTime - commandStartTime) / 1000; // Convert to seconds
    const totalDuration = (commandEndTime - startTime) / 1000; // Convert to seconds
    console.log(`creating version ${i+1} of ${NUM_OF_VERSIONS} (took ${commandDuration.toFixed(2)}s, average ${(totalDuration / (i + 1)).toFixed(2)}s)`);
  }
  const endTime = Date.now();
  const totalDuration = (endTime - startTime) / 1000; // Convert to seconds
  console.log(`total time to create ${NUM_OF_VERSIONS} versions: ${totalDuration.toFixed(2)}s`);

  // validate that the versions created via the CLI are visible via the CLI with pagination
  const versionCheckConfig: VersionCheckConfig = {
    testAppSlug: APP_SLUG,
    testNamespace: NAMESPACE,
    testDefaultPageSize: DEFAULT_PAGE_SIZE,
    testLatestSequence: LATEST_SEQUENCE,
  };

  checkVersions(versionCheckConfig);

  // validate that the versions created via the CLI are visible via the UI
  console.log("validating the first page of versions via the UI");
  await page.getByRole('link', { name: 'Version history' }).click();
  await expect(page.getByText('All versions')).toBeVisible();
  // should be 251 (new version available at top of page)
  await expect(page.getByTestId('available-updates-card').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('available-updates-card').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE}`);
  // should be 251 (first entry in all versions card)
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE}`);
  // should be 250 (second entry in all versions card)
  await expect(page.getByTestId('version-history-row-1').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('version-history-row-1').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - 1}`);

  // should be 232 (last entry in first page of all versions card)
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - 19}`);
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 1 - 20 of ${NUM_OF_VERSIONS + 1}`);

  console.log("validating the second page of versions via the UI");
  await page.getByTestId('pager-next').click();
  // should be 21 - 40 of 252
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 21 - 40 of ${NUM_OF_VERSIONS + 1}`);
  await expect(page.getByTestId('available-updates-card').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE}`);
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - DEFAULT_PAGE_SIZE}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - DEFAULT_PAGE_SIZE - 19}`);

  // make sure that going further forward and backward works
  console.log("validating going further forward and backward in the UI");
  await page.getByTestId('pager-next').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 41 - 60 of ${NUM_OF_VERSIONS + 1}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - (DEFAULT_PAGE_SIZE * 2) - 19}`);
  await page.getByTestId('pager-next').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 61 - 80 of ${NUM_OF_VERSIONS + 1}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - (DEFAULT_PAGE_SIZE * 3) - 19}`);
  await page.getByTestId('pager-prev').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 41 - 60 of ${NUM_OF_VERSIONS + 1}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - (DEFAULT_PAGE_SIZE * 2) - 19}`);
  
  // go back to the dashboard to reset things
  await page.getByRole('link', { name: 'Dashboard' }).click();
  await expect(page.getByTestId('dashboard-app-status')).toBeVisible();

  // enter the version history page and set the page size to 100
  console.log("validating setting the page size to 100 via the UI");
  await page.getByRole('link', { name: 'Version history' }).click();
  await expect(page.getByText('New version available')).toBeVisible();
  await page.getByRole('combobox').selectOption('100');
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 1 - 100 of ${NUM_OF_VERSIONS + 1}`);
  await page.getByTestId('pager-next').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 101 - 200 of ${NUM_OF_VERSIONS + 1}`);
  // the bottom of the second page should be the 199th-from-latest version (52)
  await expect(page.getByTestId('version-history-row-99').getByTestId('version-sequence')).toContainText(`Sequence ${LATEST_SEQUENCE - (100 * 1) - 99}`);
  
  console.log("validating deploying the a specific older version via the UI");
  await page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByRole('button', { name: 'Deploy' }).click();
  // the first version on the second page should be the 100th-from-latest version (151)
  await expect(page.getByLabel('Confirm deployment').getByRole('paragraph')).toContainText(`(Sequence ${LATEST_SEQUENCE - 100})?`);
  await page.getByRole('button', { name: 'Yes, deploy' }).click();
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-status').locator('span')).toContainText('Currently deployed version');

  // ensure that the deployed version and latest version are both visible on the dashboard
  await page.getByRole('link', { name: 'Dashboard' }).click();
  await expect(page.getByTestId('page-dashboard')).toContainText(`Sequence ${LATEST_SEQUENCE - 100}`);
  await expect(page.getByTestId('page-dashboard')).toContainText(`Sequence ${LATEST_SEQUENCE}`);
});

interface VersionCheckConfig {
  testAppSlug: string;
  testNamespace: string;
  testDefaultPageSize: number;
  testLatestSequence: number;
}

export function checkVersions(config: VersionCheckConfig): void {
  const {
    testAppSlug,
    testNamespace,
    testDefaultPageSize,
    testLatestSequence
  } = config;

  // Check default values
  let getVersionsCommand = `kubectl kots get versions ${testAppSlug} -n ${testNamespace} | awk 'NR>1' | wc -l`;
  console.log(getVersionsCommand, "\n");
  let numOfVersions = parseInt(execSync(getVersionsCommand).toString().trim());
  if (numOfVersions !== testDefaultPageSize) {
    throw new Error(`using default values: expected ${testDefaultPageSize} versions, got ${numOfVersions}.`);
  }

  // Check custom page size
  getVersionsCommand = `kubectl kots get versions ${testAppSlug} --page-size 15 -n ${testNamespace} | awk 'NR>1' | wc -l`;
  console.log(getVersionsCommand, "\n");
  numOfVersions = parseInt(execSync(getVersionsCommand).toString().trim());
  if (numOfVersions !== 15) {
    throw new Error(`using custom page size: expected 15 versions, got ${numOfVersions}.`);
  }

  // Check custom current page
  getVersionsCommand = `kubectl kots get versions ${testAppSlug} --current-page 1 -n ${testNamespace} | awk 'NR==2 {print $2}'`;
  console.log(getVersionsCommand, "\n");
  const firstSequence = parseInt(execSync(getVersionsCommand).toString().trim());
  if (firstSequence !== (testLatestSequence - (testDefaultPageSize * 1))) {
    throw new Error(`using custom current page: expected the first version sequence to be ${testLatestSequence - (testDefaultPageSize * 1)}, got ${firstSequence}.`);
  }

  // Check pin latest with custom current page
  getVersionsCommand = `kubectl kots get versions ${testAppSlug} --current-page 1 --pin-latest -n ${testNamespace} | awk 'NR==2 {print $2}'`;
  console.log(getVersionsCommand, "\n");
  const latestSequence = parseInt(execSync(getVersionsCommand).toString().trim());
  if (latestSequence !== testLatestSequence) {
    throw new Error(`using pin latest and a custom current page: expected the latest version sequence to be ${testLatestSequence}, got ${latestSequence}.`);
  }

  // Check pin latest deployable with custom current page
  getVersionsCommand = `kubectl kots get versions ${testAppSlug} --current-page 1 --pin-latest-deployable -n ${testNamespace} | awk 'NR==2 {print $2}'`;
  console.log(getVersionsCommand, "\n");
  const latestDeployableSequence = parseInt(execSync(getVersionsCommand).toString().trim());
  if (latestDeployableSequence !== testLatestSequence) {
    throw new Error(`using pin latest and a custom current page: expected the latest deployable version sequence to be ${testLatestSequence}, got ${latestDeployableSequence}.`);
  }
}