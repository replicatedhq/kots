import tar from 'tar-stream';
import gunzip from 'gunzip-maybe';
import { execSync } from 'child_process';
import { text } from 'node:stream/consumers';
import { test, expect, Page, Expect } from '@playwright/test';
import {
  login,
  uploadLicense,
  appIsReady,
} from '../shared';

test('target kots version', async ({ page, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);

  // this seems to take a really long time on okd
  test.setTimeout(10 * 60 * 1000); // 10 minutes

  await login(page);
  await uploadLicense(page, expect);
  await appIsReady(page, expect);
  await configureURLRedaction(page, expect);
  await validateSupportBundleUi(page, expect);
  await validateSupportBundleCli(page, expect);
});

async function validateSupportBundleUi(page: Page, expect: Expect) {
  await generateSupportBundleUi(page, expect);
  await validateRedaction(page, expect);
  await validateRedactionReport(page, expect);
  await validateDownloadBundle(page, expect);
  await validateGenerateBundleModal(page, expect);
  await validateSupportBundleDelete(page, expect);
}

async function validateSupportBundleCli(page: Page, expect: Expect) {
  await generateSupportBundleCli(page, expect);
  await validateRedaction(page, expect);
  await validateRedactionReport(page, expect);
  await validateDownloadBundle(page, expect);
  await validateGenerateBundleModal(page, expect);
  await validateSupportBundleDelete(page, expect);
}

async function generateSupportBundleUi(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Troubleshoot" }).click();

  await page.getByTestId("btn-analyze-app").click();

  await expect(page.getByTestId("page-support-bundle-analysis")).toBeVisible({ timeout: 30 * 1000 }); // 30 seconds
  await expect(page.getByTestId("support-bundle-analysis-progress-container")).toBeVisible();
  // this seems to take a really long time on okd
  await expect(page.getByTestId("support-bundle-analysis-progress-container")).not.toBeVisible({ timeout: 8 * 60 * 1000 }); // 8 minutes

  await expect(page.getByTestId("support-bundle-analysis-bundle-insights-tab")).toBeVisible();
  await page.getByTestId("support-bundle-analysis-bundle-insights-tab").click();
  await expect(page.getByTestId("support-bundle-analysis-bundle-insights")).toContainText("Only show errors and warnings");
}

async function generateSupportBundleCli(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Troubleshoot" }).click();

  await page.getByTestId("link-generate-support-bundle-command").click();

  const code = page.getByTestId("code-snippet-support-bundle-command");
  await expect(code).toBeVisible();
  await expect(code).toContainText("kubectl support-bundle");
  await code.getByTestId("btn-copy-code-snippet").click();
  const handle = await page.evaluateHandle(() => navigator.clipboard.readText());
  const clipboardContent = await handle.jsonValue();
  expect(clipboardContent).toContain("kubectl support-bundle");

  await runSupportBundleCommand(clipboardContent);

  // this seems to take a really long time on okd
  await expect(page.getByTestId("support-bundle-analysis-bundle-insights-tab")).toBeVisible({ timeout: 8 * 60 * 1000 }); // 8 minutes
  await page.getByTestId("support-bundle-analysis-bundle-insights-tab").click();
  await expect(page.getByTestId("support-bundle-analysis-bundle-insights")).toContainText("Only show errors and warnings");
}

async function validateRedaction(page: Page, expect: Expect) {
  await page.getByTestId("support-bundle-analysis-file-inspector-tab").click();
  await expect(page.getByTestId("support-bundle-analysis-file-inspector")).toBeVisible();

  const fileTree = page.getByTestId("support-bundle-analysis-file-tree");
  await expect(fileTree).toBeVisible();

  await fileTree.getByTestId("cluster-info").check();
  await fileTree.getByTestId("cluster-info/cluster_version.json").click();

  await expect(page.getByTestId("support-bundle-analysis-file-inspector-editor")).toContainText("***HIDDEN***");

  // assert that there are 2 redactions in the editor
  const fileText = await page.getByTestId("support-bundle-analysis-file-inspector-editor").textContent();
  const redactions = (fileText?.match(/HIDDEN/g) || []).length;
  expect(redactions).toBe(2);
}

async function validateRedactionReport(page: Page, expect: Expect) {
  await page.getByTestId("support-bundle-analysis-redactor-report-tab").click();

  const report = page.getByTestId("support-bundle-analysis-redactor-report");
  await expect(report).toBeVisible();

  // validate the redactor row has redactions in files
  const redactor = report.getByTestId("support-bundle-analysis-redactor-report-row-IP Addresses.regex.0");
  await expect(redactor).toContainText("IP Addresses.regex.0");
  await expect(redactor).toContainText(/[1-9][0-9]* redactions/);
  await expect(redactor).toContainText(/[1-9][0-9]* files/);

  // validate the details contains files and follow the link to the file tree
  await redactor.getByTestId("link-redactor-report-row-details").click();

  const file = redactor.getByTestId("link-redactor-report-details-file-cluster-resources/nodes.json");
  await expect(file).toBeVisible();
  await file.getByTestId("link-redactor-report-details-go-to-file").click();

  await expect(page.getByTestId("support-bundle-analysis-file-inspector")).toBeVisible();
  await expect(page.getByTestId("support-bundle-analysis-file-tree")).toBeVisible();
  await expect(page.getByTestId("support-bundle-analysis-file-inspector-editor")).toContainText("***HIDDEN***");
  await expect(page.getByTestId("support-bundle-analysis-file-inspector-redaction-pager")).toContainText(/Redaction [0-9]+ of [0-9]+/);
}

async function validateDownloadBundle(page: Page, expect: Expect) {
  const downloadPromise = page.waitForEvent('download', { timeout: 1 * 60 * 1000 }); // 1 minute

  await page.getByTestId("support-bundle-analysis-download-bundle").click();

  const download = await downloadPromise;
  const stream = await download.createReadStream();

  const extract = tar.extract();
  stream.pipe(gunzip()).pipe(extract);

  await validateBundleFiles(expect, extract);

  await page.getByTestId("link-support-bundle-analysis-back").click();
}

async function validateBundleFiles(expect: Expect, extract: tar.Extract) {
  let foundAnalysisFile = false;
  let foundVersionFile = false;
  let foundClusterResourcesDir = false;

  for await (const entry of extract) {
    const path = entry.header.name;
    const parts = path.split("/");

    if (`${parts[1]}/${parts[2]}` === "cluster-info/cluster_version.json") {
      const content = await text(entry);
      const redactions = (content.match(/HIDDEN/g) || []).length;
      expect(redactions, "Expected to find 2 redactions in cluster-info/cluster_version.json").toBe(2);
    } else if (parts[1] === "analysis.json") {
      foundAnalysisFile = true;
    } else if (parts[1] === "version.yaml") {
      foundVersionFile = true;
    } else if (parts[1] === "cluster-resources") {
      foundClusterResourcesDir = true;
    }

    entry.resume()
  }

  expect(foundAnalysisFile, "Expected to find analysis.json file in the bundle").toBe(true);
  expect(foundVersionFile, "Expected to find version.yaml file in the bundle").toBe(true);
  expect(foundClusterResourcesDir, "Expected to find cluster-resources directory in the bundle").toBe(true);
}

async function validateGenerateBundleModal(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Troubleshoot" }).click();

  await page.getByTestId("link-support-bundle-generate").click();

  const modal = page.getByTestId("modal-generate-support-bundle");
  await expect(modal).toBeVisible();

  await modal.getByTestId("link-generate-support-bundle-command").click();

  const code = page.getByTestId("code-snippet-support-bundle-command");
  await expect(code).toBeVisible();
  await expect(code).toContainText("kubectl support-bundle");
  await code.getByTestId("btn-copy-code-snippet").click();
  const handle = await page.evaluateHandle(() => navigator.clipboard.readText());
  const clipboardContent = await handle.jsonValue();
  expect(clipboardContent).toContain("kubectl support-bundle");

  const dropzone = modal.getByTestId("dropzone-support-bundle-file");
  await expect(dropzone).toBeVisible();
  await expect(dropzone).toContainText("Drag your bundle here or");

  await modal.getByTestId("btn-generate-support-bundle-modal-close").click();
  await expect(modal).not.toBeVisible();
}

async function validateSupportBundleDelete(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Troubleshoot" }).click();

  const row = page.getByTestId("support-bundle-row-0");
  await expect(row).toBeVisible();
  await row.getByTestId("btn-support-bundle-delete").click();

  // validate the delete undo button works
  let toast = page.getByTestId("toast");
  await expect(toast).toBeVisible();
  await expect(toast).toContainText("Deleting bundle collected on");
  await toast.getByTestId("btn-support-bundle-delete-undo").click();
  await expect(toast).not.toBeVisible();
  await expect(row).toBeVisible();

  // Work around a ui bug in toast where if you cancel and retry too quickly, it will not make the
  // subsequent request. Toast timeout is 7 seconds.
  await page.waitForTimeout(8 * 1000); // 8 seconds

  // validate the delete button works
  await row.getByTestId("btn-support-bundle-delete").click();
  toast = page.getByTestId("toast");
  await expect(toast).toBeVisible();
  await expect(toast).toContainText("Deleting bundle collected on");
  await expect(toast).not.toBeVisible({ timeout: 30 * 1000 }); // 30 seconds
  await expect(row).not.toBeVisible();
}

async function configureURLRedaction(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Troubleshoot" }).click();

  await page.getByTestId("link-configure-redaction").click();

  const modal = page.getByTestId("modal-configure-redaction");
  await expect(modal).toBeVisible();
  await expect(modal.locator('.Loader')).not.toBeVisible({ timeout: 15 * 1000 }); // 15 seconds

  await expect(modal.getByTestId("link-redactors-link-to-a-spec")).toBeVisible();
  await modal.getByTestId("link-redactors-link-to-a-spec").click();

  await expect(modal.getByTestId("input-redactor-uri")).toBeVisible();
  await modal.getByTestId("input-redactor-uri").fill("https://raw.githubusercontent.com/replicatedhq/kots/master/testim/testim-redactor-spec.yaml");

  await modal.getByTestId("btn-redactor-modal-save").click();
  await modal.getByTestId("btn-redactor-modal-close").click();
  await expect(modal).not.toBeVisible();
}

async function runSupportBundleCommand(command: string) {
  let fixedCommand = command.replace("\n", "").replace("bashkubectl", "bash && yes | kubectl").trim();
  fixedCommand += ` --interactive=false --allow-insecure-connections`;
  execSync(fixedCommand, {stdio: 'inherit'});
}
