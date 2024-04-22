import { test, expect } from '@playwright/test';
import * as parse_duration from 'parse-duration';
import { retry } from 'ts-retry';
import { login, uploadLicense } from '../shared';

const { execSync } = require("child_process");

test('backup and restore', async ({ page }) => {
  test.setTimeout(10 * 60 * 1000); // 10 minutes
  await login(page);
  await uploadLicense(page, expect);
  await expect(page.locator('#app')).toContainText('Configure Backup and Restore');
  await page.locator('#smtp_hostname-group').getByRole('textbox').click();
  await page.locator('#smtp_hostname-group').getByRole('textbox').fill('hostname');
  await page.locator('#smtp_username-group').getByRole('textbox').click();
  await page.locator('#smtp_username-group').getByRole('textbox').fill('username');
  await page.locator('input[type="password"]').click();
  await page.locator('input[type="password"]').fill('password');
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Ready', { timeout: 15000 });
  await page.getByText('Snapshots').click();
  await expect(page.locator('#app')).toContainText('No snapshots yet');
  await page.getByRole('button', { name: 'Start a snapshot' }).click();
  await expect(page.locator('#app')).toContainText('In Progress');
  await expect(page.locator('#app')).toContainText('Completed');
  await page.locator('svg.icons.clickable[data-tip="Restore from this backup"]').click();
  await page.getByText('Restore admin console', { exact: true }).click();

  const backupName = await page.locator('.card-item-title').textContent();
  const restoreAdminConsoleCmd = `kubectl kots restore --from-backup ${backupName} --exclude-apps`;
  console.log(restoreAdminConsoleCmd, "\n");
  execSync(restoreAdminConsoleCmd, {stdio: 'inherit'});

  // validate the only the admin console was restored
  const getKotsadmPodAgeCommand = `kubectl get pod -l app=kotsadm -n ${process.env.NAMESPACE} | awk 'NR>1 {print $5}'`;
  console.log(getKotsadmPodAgeCommand, "\n");
  let kotsadmPodAge = parse_duration(execSync(getKotsadmPodAgeCommand).toString());

  const getAppPodAgeCommand = `kubectl get pod -l app=example,component=nginx -n ${process.env.NAMESPACE} | awk 'NR>1 {print $5}'`;
  console.log(getAppPodAgeCommand, "\n");
  let appPodAge = parse_duration(execSync(getAppPodAgeCommand).toString());

  // app pod should be older than kotsadm pod
  let ageDiff = appPodAge! - kotsadmPodAge!;
  console.log(`application pod is ${ageDiff}ms older than the kotsadm pod`);
  if (ageDiff < 5000) {
    throw new Error("Expected the application pod to be older than the kotsadm pod");
  }

  const restoreAppCommand = `kubectl kots restore --from-backup ${backupName} --exclude-admin-console`;
  console.log(restoreAppCommand, "\n");
  execSync(restoreAppCommand, {stdio: 'inherit'});

  await retry(
    () => {
      const getAppPodCommand = `kubectl get pod -l app=example,component=nginx -n ${process.env.NAMESPACE} | grep example-nginx`;
      console.log(getAppPodCommand, "\n");
      execSync(getAppPodCommand, {stdio: 'inherit'});
    },
    { delay: 1000, maxTry: 10 }
  );

  // validate that only the app was restored
  console.log(getAppPodAgeCommand, "\n");
  appPodAge = parse_duration(execSync(getAppPodAgeCommand).toString());

  console.log(getKotsadmPodAgeCommand, "\n");
  kotsadmPodAge = parse_duration(execSync(getKotsadmPodAgeCommand).toString());

  // kotsadm pod should be older than app pod
  ageDiff = kotsadmPodAge! - appPodAge!;
  console.log(`kotsadm pod is ${ageDiff}ms older than the application pod`);
  if (ageDiff < 5000) {
    throw new Error("Expected the kotsadm pod to be older than the application pod");
  }
});
