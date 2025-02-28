import { Page, Expect, Locator } from '@playwright/test';

import { RegistryInfo } from './cli';
import { APP_SLUG } from './constants';

export const validateViewFiles = async (
  page: Page,
  expect: Expect,
  channelId: string,
  channelName: string,
  customerName: string,
  licenseId: string,
  isAirgapped: boolean,
  registryInfo: RegistryInfo
) => {
  // resize page so that the entire editor content is visible for text detection
  await page.setViewportSize({ width: 3840, height: 2160 });

  await page.locator('.NavItem').getByText('Application', { exact: true }).click();
  await page.getByRole('link', { name: 'View files', exact: true }).click();

  const viewFilesPage = page.getByTestId('view-files-page');
  await expect(viewFilesPage).toBeVisible();
  await expect(viewFilesPage.getByTestId('file-editor-empty-state')).toBeVisible();

  const fileTree = viewFilesPage.getByTestId('file-tree');
  await expect(fileTree).toBeVisible({ timeout: 15000 });

  // Validate upstream directory
  await selectFile(page, fileTree, '/upstream');

  await selectFile(page, fileTree, '/upstream/backup.yaml');
  const editor = viewFilesPage.getByTestId('file-editor');
  await expect(editor).toContainText('kind: Backup');

  await selectFile(page, fileTree, '/upstream/postgresql.yaml');
  await expect(editor).toContainText('kind: HelmChart');

  await selectFile(page, fileTree, '/upstream/userdata');
  await selectFile(page, fileTree, '/upstream/userdata/config.yaml');
  await expect(editor).toContainText('kind: ConfigValues');

  await selectFile(page, fileTree, '/upstream/userdata/identityconfig.yaml');
  await expect(editor).toContainText('kind: IdentityConfig');

  await selectFile(page, fileTree, '/upstream/userdata/installation.yaml');
  const installationYAML = await getEditorText(editor);
  expect(installationYAML).toContain('kind: Installation');
  expect(installationYAML).toContain(channelId);
  expect(installationYAML).toContain(channelName);

  await selectFile(page, fileTree, '/upstream/userdata/license.yaml');
  const licenseYAML = await getEditorText(editor);
  expect(licenseYAML).toContain('kind: License');
  expect(licenseYAML).toContain(channelId);
  expect(licenseYAML).toContain(channelName); 
  expect(licenseYAML).toContain(customerName);
  expect(licenseYAML).toContain(licenseId);

  await selectFile(page, fileTree, '/upstream');
  await expect(fileTree.getByTestId('/upstream/backup.yaml')).not.toBeVisible();

  // Validate base directory
  await selectFile(page, fileTree, '/base');
  await selectFile(page, fileTree, '/base/charts');
  await selectFile(page, fileTree, '/base/charts/postgresql');
  await selectFile(page, fileTree, '/base/charts/postgresql/statefulset.yaml');

  const postgresqlYAML = await getEditorText(editor);
  expect(postgresqlYAML).toContain('kind: StatefulSet');
  expect(postgresqlYAML).toContain('name: postgresql-postgresql');

  await selectFile(page, fileTree, '/base');
  await expect(fileTree.getByTestId('/base/charts')).not.toBeVisible();

  // Validate overlays directory
  await selectFile(page, fileTree, '/overlays');
  await selectFile(page, fileTree, '/overlays/downstreams');
  await selectFile(page, fileTree, '/overlays/downstreams/this-cluster');
  await selectFile(page, fileTree, '/overlays/downstreams/this-cluster/kustomization.yaml');
  
  await selectFile(page, fileTree, '/overlays/downstreams');
  await expect(fileTree.getByTestId('/overlays/downstreams/this-cluster')).not.toBeVisible();
  
  await selectFile(page, fileTree, '/overlays/midstream');
  await selectFile(page, fileTree, '/overlays/midstream/pullsecrets.yaml');
  
  const pullSecretsYAML = await getEditorText(editor);
  const pullSecretsCount = (pullSecretsYAML!.match(/name: qakotsregression-registry/g) || []).length;
  expect(pullSecretsCount).toBeGreaterThanOrEqual(3);
  
  await selectFile(page, fileTree, '/overlays/midstream/kustomization.yaml');

  const kustomizationYAML = await getEditorText(editor);
  expect(kustomizationYAML).toContain('kind: Kustomization');
  expect(kustomizationYAML).toContain('- pullsecrets.yaml');
  expect(kustomizationYAML).toContain('- secret.yaml');
  expect(kustomizationYAML).toContain('transformers:');
  expect(kustomizationYAML).toContain('- backup-label-transformer.yaml');

  if (isAirgapped) {
    expect(kustomizationYAML).toContain(`newName: ${registryInfo.ip}/${APP_SLUG}/qa-kots`);
    expect(kustomizationYAML).toContain(`newName: ${registryInfo.ip}/${APP_SLUG}/postgresql`);
    expect(kustomizationYAML).toContain(`newName: ${registryInfo.ip}/${APP_SLUG}/busybox`);
    expect(kustomizationYAML).toContain(`newName: ${registryInfo.ip}/${APP_SLUG}/qa-mysql`);
    expect(kustomizationYAML).toContain(`newName: ${registryInfo.ip}/${APP_SLUG}/nginx`);
  } else {
    expect(kustomizationYAML).toContain(`newName: proxy.replicated.com/proxy/${APP_SLUG}/429114214526.dkr.ecr.us-east-1.amazonaws.com/qa-kots`);
    expect(kustomizationYAML).toContain(`newName: proxy.replicated.com/proxy/${APP_SLUG}/repldev/qa-mysql`);
    expect(kustomizationYAML).toContain(`newName: proxy.replicated.com/proxy/${APP_SLUG}/quay.io/replicatedcom/qa-kots`);
    expect(kustomizationYAML).toContain(`newName: proxy.replicated.com/proxy/${APP_SLUG}/gcr.io/replicated-qa/dnsutils`);
  }

  // resize page back to default size
  await page.setViewportSize({ width: 1280, height: 720 });
};

const selectFile = async (page: Page, fileTree: Locator, file: string) => {
  await fileTree.getByTestId(file).click();
  await page.waitForTimeout(500); // a small delay to ensure the ui has time to update
};

const getEditorText = async (editor: Locator): Promise<string> => {
  const text = await editor.textContent() || '';
  return text.replace(/\s+/g, ' '); // fix white space encoding
};
