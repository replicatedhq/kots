import { Page, Expect, Locator } from '@playwright/test';

import { runCommand } from './cli';
import {
  SSH_TO_WORKER,
  SSH_TO_JUMPBOX
} from './constants';

export const joinWorkerNode = async (page: Page, expect: Expect) => {
  await expect(page.getByTestId('get-started-sidebar')).not.toBeVisible({ timeout: 15000 });
  await page.locator('.NavItem').getByText('Cluster Management', { exact: true }).click();
  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  await page.getByRole('button', { name: 'Add a node', exact: true }).click();
  await page.getByTestId('secondary-node-radio').click();

  const addNodeSnippet = page.getByTestId('add-node-command');
  await expect(addNodeSnippet).toBeVisible({ timeout: 30000 });
  let addNodeCommand = await addNodeSnippet.locator(".react-prism.language-bash").textContent();
  expect(addNodeCommand).not.toBeNull();

  addNodeCommand = `${addNodeCommand} yes`; // for the nightly release prompt
  addNodeCommand = `${SSH_TO_WORKER} '${addNodeCommand}'`; // run on the worker node
  addNodeCommand = `${SSH_TO_JUMPBOX} "${addNodeCommand}" &`; // via the jumpbox in the background

  runCommand(addNodeCommand);
};

export const validateClusterManagement = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Cluster Management', { exact: true }).click();

  const allNodes = page.getByTestId('all-nodes-list');
  await expect(allNodes).toBeVisible({ timeout: 15000 });

  const firstNode = allNodes.getByTestId('kurl-node-row-0');
  await validateNodeRow(page, expect, firstNode);

  const secondNode = allNodes.getByTestId('kurl-node-row-1');
  await validateNodeRow(page, expect, secondNode);

  // manually expire join cert
  runCommand(`kubectl patch cm kurl-config -n kube-system -p '{"data":{"upload_certs_expiration":"2000-01-01T00:00:00Z"}}'`);

  const addNodeSnippet = page.getByTestId('add-node-command');
  await expect(addNodeSnippet).not.toBeVisible();

  await page.getByRole('button', { name: 'Add a node', exact: true }).click();
  await page.getByTestId('primary-node-radio').click();
  await expect(page.getByTestId('add-node-command-loader')).not.toBeVisible({ timeout: 30000 });
  await expect(addNodeSnippet).toBeVisible();

  await page.getByTestId('secondary-node-radio').click();
  await expect(page.getByTestId('add-node-command-loader')).not.toBeVisible({ timeout: 30000 });
  await expect(addNodeSnippet).toBeVisible();
};

const validateNodeRow = async (page: Page, expect: Expect, nodeRow: Locator) => {
  await expect(nodeRow).toBeVisible();
  await expect(nodeRow.getByTestId('node-name')).toBeVisible();
  await expect(nodeRow.getByTestId('node-status')).toHaveText('Connected');

  const nodePods = nodeRow.getByTestId('node-pods');
  await expect(nodePods).toBeVisible();
  expect(extractNumber(await nodePods.textContent())).toBeGreaterThan(0);

  const nodeCpu = nodeRow.getByTestId('node-cpu');
  await expect(nodeCpu).toBeVisible();
  expect(extractNumber(await nodeCpu.textContent())).toBeGreaterThan(0);

  const nodeMemory = nodeRow.getByTestId('node-memory');
  await expect(nodeMemory).toBeVisible();
  expect(extractNumber(await nodeMemory.textContent())).toBeGreaterThan(0);

  await expect(nodeRow).toContainText('No Disk Pressure');
  await expect(nodeRow).toContainText('No CPU Pressure');
  await expect(nodeRow).toContainText('No Memory Pressure');

  await nodeRow.getByRole('button', { name: 'Drain node' }).click();
  const drainNodeModal = page.getByTestId('drain-node-modal');
  await expect(drainNodeModal).toBeVisible();
  await expect(drainNodeModal.getByRole('button', { name: /Drain/ })).toBeVisible();
  await drainNodeModal.getByRole('button', { name: 'Cancel' }).click();
  await expect(drainNodeModal).not.toBeVisible();
}

const extractNumber = (text: string): number => {
  const match = text.match(/\d+(\.\d+)?/);
  if (!match || match.length < 1) {
    throw new Error(`Number not found in text "${text}"`);
  }
  const number = parseFloat(match[0]);
  if (isNaN(number)) {
    throw new Error(`Not a number "${match[0]}"`);
  }
  return number;
};
