import { Page, Expect } from '@playwright/test';

export const validateDashboardInfo = async (page: Page, expect: Expect, isAirgapped: boolean) => {
  await page.locator('.NavItem').getByText('Application', { exact: true }).click();
  await page.getByRole('link', { name: 'Dashboard', exact: true }).click();

  await expect(page.getByTestId("dashboard-app-icon")).toBeVisible();
  await expect(page.getByTestId("dashboard-app-name")).toBeVisible();
  await expect(page.getByTestId("dashboard-app-status")).toHaveText("Ready", { timeout: 210000 });
  await expect(page.getByTestId("dashboard-edit-config")).toBeVisible();
  if (!isAirgapped) {
    await expect(page.getByTestId("dashboard-app-link")).toBeVisible();
  }

  const versionCard = page.getByTestId("dashboard-version-card");
  await expect(versionCard).toBeVisible();
  await expect(versionCard.getByTestId("current-version-status")).toHaveText('Currently deployed version', { timeout: 30000 });
  await expect(versionCard.getByText("See all versions")).toBeVisible();

  const licenseCard = page.getByTestId("dashboard-license-card");
  await expect(licenseCard).toBeVisible();
  await expect(licenseCard.getByText(isAirgapped ? "Upload license" : "Sync license")).toBeVisible();
  await expect(licenseCard.getByText("See license details")).toBeVisible();

  const snapshotsCard = page.getByTestId("dashboard-snapshots-card");
  await expect(snapshotsCard).toBeVisible();
  await expect(snapshotsCard.getByText("Snapshot settings")).toBeVisible();
  await expect(snapshotsCard.getByText("Start snapshot")).toBeVisible();
  await expect(snapshotsCard.getByText("See all snapshots")).toBeVisible();
};

export const validateDashboardAutomaticUpdates = async (page: Page, expect: Expect) => {
  await page.getByText('Configure automatic updates').click();
  const automaticUpdatesModal = page.getByTestId('automatic-updates-modal');
  await expect(automaticUpdatesModal).toBeVisible();

  await automaticUpdatesModal.locator(".replicated-select__control").click();
  await page.waitForTimeout(1000);
  await automaticUpdatesModal.locator(".replicated-select__option").getByText("Custom", { exact: true }).click();
  await page.waitForTimeout(1000);
  await expect(automaticUpdatesModal.getByTestId("update-checker-spec")).toHaveValue("0 2 * * WED,SAT");
  await expect(automaticUpdatesModal).toContainText("At 02:00 AM, only on Wednesday and Saturday");

  await automaticUpdatesModal.getByTestId("update-checker-spec").click();
  await page.waitForTimeout(1000);
  await automaticUpdatesModal.getByTestId("update-checker-spec").fill("0 2 * * SUN");
  await expect(automaticUpdatesModal).toContainText("At 02:00 AM, only on Sunday");
  await automaticUpdatesModal.getByRole("button", { name: "Update", exact: true }).click();
  await expect(automaticUpdatesModal).not.toBeVisible();
};

export const validateDashboardGraphs = async (page: Page, expect: Expect) => {
  const graphsCard = page.getByTestId("dashboard-graphs-card");
  await expect(graphsCard).toBeVisible();
  await graphsCard.getByTestId("prometheus-endpoint").click();
  await graphsCard.getByTestId("prometheus-endpoint").fill("http://prometheus-k8s.monitoring.svc.cluster.local:9090");
  await graphsCard.getByRole('button', { name: 'Save' }).click();

  await expect(graphsCard).toContainText("Disk Usage");
  await expect(graphsCard).toContainText("CPU Usage");
  await expect(graphsCard).toContainText("Memory Usage");

  const diskUsageGraph = graphsCard.getByTestId("graph-disk-usage");
  await expect(diskUsageGraph).toBeVisible();
  await expect(diskUsageGraph).toContainText("Used");
  await expect(diskUsageGraph).toContainText("Available");

  const cpuUsageGraph = graphsCard.getByTestId("graph-cpu-usage");
  await expect(cpuUsageGraph).toBeVisible();
  await expect(cpuUsageGraph).toContainText("kotsadm-rqlite");

  const memoryUsageGraph = graphsCard.getByTestId("graph-memory-usage");
  await expect(memoryUsageGraph).toBeVisible();
  await expect(memoryUsageGraph).toContainText("kotsadm-rqlite");
};