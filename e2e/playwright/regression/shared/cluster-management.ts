import { Page, Expect } from '@playwright/test';

import { runCommand } from './cli';
import { SSH_TO_WORKER } from './constants';

export const joinWorkerNode = async (page: Page, expect: Expect) => {
  await page.locator('.NavItem').getByText('Cluster Management', { exact: true }).click();

  await expect(page.locator('.Loader')).not.toBeVisible({ timeout: 15000 });

  await page.getByRole('button', { name: 'Add a node', exact: true }).click();
  await page.getByTestId('secondary-node-radio').click();

  const addNodeSnippet = page.getByTestId('add-node-command');
  await expect(addNodeSnippet).toBeVisible({ timeout: 30000 });
  let addNodeCommand = await addNodeSnippet.locator(".react-prism.language-bash").textContent();
  expect(addNodeCommand).not.toBeNull();

  addNodeCommand = addNodeCommand.replace(/\\/g, "");
  addNodeCommand = `${addNodeCommand} yes`; // for the nightly release prompt
  addNodeCommand = `${SSH_TO_WORKER} "${addNodeCommand}" &`; // run this in the background

  runCommand(addNodeCommand, true);
};
