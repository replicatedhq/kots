import { test, expect, Page } from '@playwright/test';
import { login, uploadLicense } from '../shared';
import { execSync } from 'child_process';

test('version history pagination', async ({ page }) => {
  test.slow();
  test.setTimeout(300000); // 300 seconds, 5 minutes
  const testAppSlug = "version-history-pagination";
  const testNamespace = "version-history-pagination";
  const testNumOfVersions = 251;
  const testLatestSequence = 251;
  const testDefaultPageSize = 20;
  await login(page);
  await uploadLicense(page, expect, "version-history-pagination.yaml");
  await expect(page.locator('#app')).toContainText('Configure Version History Pagination', { timeout: 15000 });
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Currently deployed version', { timeout: 15000 });
  await expect(page.locator('#app')).toContainText('Ready', { timeout: 30000 });
  
  // create many versions via the CLI
  const startTime = Date.now();
  for (let i = 0; i < testNumOfVersions; i++) {
    const commandStartTime = Date.now();
    execSync(`kubectl kots set config ${testAppSlug} -n ${testNamespace} item_1="version-${i}"`);
    const commandEndTime = Date.now();
    const commandDuration = (commandEndTime - commandStartTime) / 1000; // Convert to seconds
    const totalDuration = (commandEndTime - startTime) / 1000; // Convert to seconds
    console.log(`creating version ${i+1} of ${testNumOfVersions} (took ${commandDuration.toFixed(2)}s, average ${(totalDuration / (i + 1)).toFixed(2)}s)`);
  }
  const endTime = Date.now();
  const totalDuration = (endTime - startTime) / 1000; // Convert to seconds
  console.log(`total time to create ${testNumOfVersions} versions: ${totalDuration.toFixed(2)}s`);

  // validate that the versions created via the CLI are visible via the CLI with pagination
  const versionCheckConfig: VersionCheckConfig = {
    testAppSlug: testAppSlug,
    testNamespace: testNamespace,
    testDefaultPageSize: testDefaultPageSize,
    testLatestSequence: testLatestSequence,
  };

  checkVersions(versionCheckConfig);

  // validate that the versions created via the CLI are visible via the UI
  await page.getByRole('link', { name: 'Version history' }).click();
  await expect(page.getByText('All versions')).toBeVisible();
  // should be 251 (new version available at top of page)
  await expect(page.getByTestId('available-updates-card').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('available-updates-card').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence}`);
  // should be 251 (first entry in all versions card)
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence}`);
  // should be 250 (second entry in all versions card)
  await expect(page.getByTestId('version-history-row-1').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('version-history-row-1').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - 1}`);

  // should be 232 (last entry in first page of all versions card)
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toBeVisible();
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - 19}`);

  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 1 - 20 of ${testNumOfVersions + 1}`);
  await page.getByText('Next').click();
  // should be 21 - 40 of 252
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 21 - 40 of ${testNumOfVersions + 1}`);
  await expect(page.getByTestId('available-updates-card').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence}`);
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - testDefaultPageSize}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - testDefaultPageSize - 19}`);

  // make sure that going further forward and backward works
  await page.getByText('Next').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 41 - 60 of ${testNumOfVersions + 1}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - (testDefaultPageSize * 2) - 19}`);
  await page.getByText('Next').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 61 - 80 of ${testNumOfVersions + 1}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - (testDefaultPageSize * 3) - 19}`);
  await page.getByText('Prev').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 41 - 60 of ${testNumOfVersions + 1}`);
  await expect(page.getByTestId('version-history-row-19').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - (testDefaultPageSize * 2) - 19}`);
  
  // go back to the dashboard to reset things
  await page.getByRole('link', { name: 'Dashboard' }).click();
  await expect(page.getByTestId('app-status-status')).toBeVisible();

  // enter the version history page and set the page size to 100
  await page.getByRole('link', { name: 'Version history' }).click();
  await expect(page.getByText('New version available')).toBeVisible();
  await page.getByRole('combobox').selectOption('100');
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 1 - 100 of ${testNumOfVersions + 1}`);
  await page.getByText('Next').click();
  await expect(page.getByTestId('all-versions-card')).toContainText(`Showing releases 101 - 200 of ${testNumOfVersions + 1}`);
  // the bottom of the second page should be the 199th-from-latest version (52)
  await expect(page.getByTestId('version-history-row-99').getByTestId('version-sequence')).toContainText(`Sequence ${testLatestSequence - (100 * 1) - 99}`);
  await page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByRole('button', { name: 'Deploy' }).click();
  // the first version on the second page should be the 100th-from-latest version (151)
  await expect(page.getByLabel('Confirm deployment').getByRole('paragraph')).toContainText(`(Sequence ${testLatestSequence - 100})?`);
  await page.getByRole('button', { name: 'Yes, deploy' }).click();
  await expect(page.getByTestId('all-versions-card').getByTestId('version-history-row-0').getByTestId('version-status').locator('span')).toContainText('Currently deployed version');

  // ensure that the deployed version and latest version are both visible on the dashboard
  await page.getByRole('link', { name: 'Dashboard' }).click();
  await expect(page.getByTestId('page-dashboard')).toContainText(`Sequence ${testLatestSequence - 100}`);
  await expect(page.getByTestId('page-dashboard')).toContainText(`Sequence ${testLatestSequence}`);
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