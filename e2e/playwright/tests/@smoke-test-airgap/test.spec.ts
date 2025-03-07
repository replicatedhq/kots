import { test, expect, Page, Expect, Locator } from "@playwright/test";
import * as constants from './constants';
import {
  login,
  logout,
  uploadLicense,
  airgapInstall,
  appIsReady,
  downloadAirgapBundle,
  filetreeSelectFile,
} from "../shared";

test("smoke test airgap", async ({ page }) => {
  test.setTimeout(5 * 60 * 1000); // 5 minutes

  await login(page);
  await uploadLicense(page, expect);
  await installAirgap(page, expect);
  await validateOnboardingConfig(page, expect);
  await validateOnboardingPreflights(page, expect);
  await appIsReady(page, expect);
  await validateDashboard(page, expect);
  await dashboardAirgapUpdate(page, expect);
  await validateConfigureUpdate(page, expect);
  await validateDeployUpdate(page, expect);
  await validateViewFiles(page, expect);
  await validateAirgapSupportBundle(page, expect);
  await logout(page, expect);
});

async function installAirgap(page: Page, expect: Expect) {
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.INSTALL_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    '/tmp/app.airgap'
  );
  await airgapInstall(page, expect, 'ttl.sh', 'admin', 'password', 'test', '/tmp/app.airgap');
}

async function validateOnboardingConfig(page: Page, expect: Expect) {
  await expect(page.locator("h3")).toContainText("My Example Config", { timeout: 30000 });
  let group = page.getByTestId("config-group-example_settings");
  let inputGroup = group.locator("#example_config_default-group");
  await expect(inputGroup.locator(".card-item-title")).toContainText("Example (with default value)");
  await expect(inputGroup.locator(".default-value-section")).toContainText("Default value: this is the default");
  inputGroup = group.locator("#example_config_required-group");
  await expect(inputGroup.locator(".card-item-title")).toContainText("Required Example (without default value)");
  await expect(inputGroup.locator(".card-item-title .required")).toContainText("Required");
  await inputGroup.getByRole("textbox").click();
  await inputGroup.getByRole("textbox").fill("some value");
  await page.getByRole("button", { name: "Continue" }).click();
};

async function validateOnboardingPreflights(page: Page, expect: Expect) {
  const resultsWrapper = page.getByTestId("preflight-results-wrapper");
  await expect(resultsWrapper.getByTestId("preflight-message-title")).toContainText("Required Kubernetes Version");
  await page.getByRole('button', { name: 'Deploy', exact: true }).click();
}

async function validateDashboard(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Dashboard" }).click();
  await expect(page.locator("#app")).toContainText("Currently deployed version");
  await expect(page.locator("#app")).toContainText("Upload new version");
  await expect(page.locator("#app")).toContainText("Redeploy");
  await expect(page.getByText("airgap-smoke-test")).toBeVisible();
  await expect(page.locator(".Dashboard--appIcon")).toBeVisible();
  await expect(page.locator("p").filter({ hasText: "License" })).toBeVisible();
}

async function dashboardAirgapUpdate(page: Page, expect: Expect) {
  await downloadAirgapBundle(
    constants.CUSTOMER_ID,
    constants.UPDATE_CHANNEL_SEQUENCE,
    constants.DOWNLOAD_PORTAL_BASE64_PASSWORD,
    '/tmp/app.airgap',
  );
  await page.setInputFiles('[data-testid="airgap-bundle-drop-zone"] input', '/tmp/app.airgap');
  let card = page.getByTestId("dashboard-version-card");
  await expect(card).toContainText("New version available", { timeout: 2 * 60 * 1000 }); // 2 minutes
  card = page.getByTestId("new-version-card");
  await expect(card).toContainText(constants.UPDATE_RELEASE_SEMVER);
  await expect(card.getByTestId("btn-version-action")).toContainText("Configure");
  await card.getByTestId("btn-version-action").click();
  await expect(page.getByTestId("config-area")).toContainText("This config is 1 version newer than the currently deployed config.");
}

async function validateConfigureUpdate(page: Page, expect: Expect) {
  page.goto("/app/airgap-smoke-test/config/1");
  let group = page.getByTestId("config-group-example_settings");
  let inputGroup = group.locator("#example_config_required_2-group");
  await expect(inputGroup.locator(".card-item-title")).toContainText("Another Required Example (without default value)");
  await expect(inputGroup.locator(".card-item-title .required")).toContainText("Required");
  await inputGroup.getByRole("textbox").click();
  await inputGroup.getByRole("textbox").fill("some other value");
  group = page.getByTestId("config-group-ports");
  await expect(group.locator("h3.card-item-title")).toContainText("Ports");
  inputGroup = group.locator("#serviceport-group");
  await expect(inputGroup.locator(".card-item-title").first()).toContainText("Service Port");
  await inputGroup.getByRole("textbox").first().click();
  await inputGroup.getByRole("textbox").first().fill("80");
  await inputGroup.getByTestId("link-add-another").click();
  await expect(inputGroup.locator(".card-item-title").nth(1)).toContainText("Service Port");
  await inputGroup.getByRole("textbox").nth(1).click();
  await inputGroup.getByRole("textbox").nth(1).fill("443");
  await page.getByTestId("btn-save-config").click();
  const nextStepModal = page.getByTestId('config-next-step-modal');
  await expect(nextStepModal).toBeVisible({ timeout: 15000 });
  await page.getByRole('button', { name: 'Go to updated version' }).click();
  await expect(nextStepModal).not.toBeVisible();
  await expect(page.getByTestId("page-app-version-history")).toBeVisible();
  const row = page.getByTestId("all-versions-card").getByTestId("version-history-row-0");
  await expect(row).toContainText(constants.UPDATE_RELEASE_SEMVER);
  await expect(row.getByTestId("version-source")).toContainText("Airgap Update");
  await expect(row.getByTestId("preflight-icon")).toContainText("Checks passed");
}

async function validateDeployUpdate(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "Version history" }).click();
  const row = page.getByTestId("all-versions-card").getByTestId("version-history-row-0");
  await expect(row).toContainText(constants.UPDATE_RELEASE_SEMVER);
  await expect(row.getByTestId("btn-version-action")).toContainText("Deploy");
  await row.getByTestId("btn-version-action").click();
  const modal = page.getByTestId('confirm-deployment-modal');
  await expect(modal).toBeVisible();
  await modal.getByRole('button', { name: 'Yes, deploy', exact: true }).click();
  await expect(modal).not.toBeVisible();
  await expect(row).toContainText("Currently deployed version");

  await page.getByTestId("console-subnav").getByRole("link", { name: "Dashboard" }).click();
  let card = page.getByTestId("current-version-card");
  await expect(card).toContainText(constants.UPDATE_RELEASE_SEMVER);
  await expect(card.getByTestId("version-source")).toContainText("Airgap Update");
  await expect(card).toContainText("Currently deployed version");
  await appIsReady(page, expect);
}

async function validateViewFiles(page: Page, expect: Expect) {
  await page.getByTestId("console-subnav").getByRole("link", { name: "View files" }).click();
  const fileTree = page.getByTestId("file-tree");
  await expect(fileTree).toBeVisible();
  await expect(fileTree.getByTestId("/upstream")).toBeVisible();
  await expect(fileTree).toContainText("upstream");
  await expect(fileTree.getByTestId("/base")).toBeVisible();
  await expect(fileTree).toContainText("base");
  await expect(fileTree.getByTestId("/overlays")).toBeVisible();
  await expect(fileTree).toContainText("overlays");
  await expect(fileTree.getByTestId("/kotsKinds")).toBeVisible();
  await expect(fileTree).toContainText("kotsKinds");
  await expect(fileTree.getByTestId("/rendered")).toBeVisible();
  await expect(fileTree).toContainText("rendered");
}

async function validateAirgapSupportBundle(page: Page, expect: Expect) {
  await generateSupportBundleUi(page, expect);
  await validateSupportBundleFileInspector(page, expect);
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

async function validateSupportBundleFileInspector(page: Page, expect: Expect) {
  await page.getByTestId("support-bundle-analysis-file-inspector-tab").click();
  await expect(page.getByTestId("support-bundle-analysis-file-inspector")).toBeVisible();
  const fileTree = page.getByTestId("support-bundle-analysis-file-tree");
  const namespace = process.env.NAMESPACE; // the namespace is not the same in the dev env as it is in ci
  await filetreeSelectFile(page, fileTree, "secrets");
  await filetreeSelectFile(page, fileTree, `secrets/${namespace}`);
  await filetreeSelectFile(page, fileTree, `secrets/${namespace}/kotsadm-airgap-smoke-test-instance-report`);
  await filetreeSelectFile(page, fileTree, `secrets/${namespace}/kotsadm-airgap-smoke-test-instance-report/report.json`);
  await filetreeSelectFile(page, fileTree, `secrets/${namespace}/kotsadm-airgap-smoke-test-preflight-report`);
  await filetreeSelectFile(page, fileTree, `secrets/${namespace}/kotsadm-airgap-smoke-test-preflight-report/report.json`);
  await filetreeSelectFile(page, fileTree, `secrets/${namespace}/replicated-instance-report`);
  await filetreeSelectFile(page, fileTree, `secrets/${namespace}/replicated-instance-report/report.json`);
}
