import { test, expect, Page } from '@playwright/test';
import { login, uploadLicense } from '../shared';

const { execSync } = require("child_process");

test('gitops install', async ({ page }) => {
  test.slow();
  await login(page);
  await uploadLicense(page, expect, "gitops.yaml");
  await expect(page.locator('#app')).toContainText('Application Configuration', { timeout: 15000 });
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Currently deployed version', { timeout: 15000 });
  // await expect(page.locator('#app')).toContainText('Ready', { timeout: 30000 });

  // the app is now installed and ready, and the real test can begin

  // create a new version with a config change
  await trivialConfig(page);

  // configure gitops
  const gitopsOwner = 'replicated-testim-kotsadm-gitops';
  const gitopsRepo = 'qakots-kotsadm-gitops';
  const githubToken = 'TODO_ADD_GITHUB_TOKEN_FROM_SECRET';
  const testAppSlug = 'gitops-bobcat';

  // generate a random branch name to use for this test
  const randomBranch = `test-${Math.random().toString(36).substring(2, 15)}`;

  // generate a random path to use for this test
  const randomPath = `/test-${Math.random().toString(36).substring(2, 15)}`;

  await page.getByText('GitOps').click();
  await expect(page.getByText('GitHub')).toBeVisible();
  await page.getByPlaceholder('owner').click();
  await page.getByPlaceholder('owner').fill(gitopsOwner);
  await page.getByPlaceholder('Repository').click();
  await page.getByPlaceholder('Repository').fill(gitopsRepo);
  await page.getByPlaceholder('main').click();
  await page.getByPlaceholder('main').fill(randomBranch);
  await page.getByPlaceholder('/path/to-deployment').click();
  await page.getByPlaceholder('/path/to-deployment').fill(randomPath);
  await page.getByRole('button', { name: 'Generate SSH key' }).click();
  await expect(page.getByText('ssh-ed25519')).toBeVisible();
  await expect(page.getByText('Copy key')).toBeVisible();
  // get the key text
  const key = page.getByText('ssh-ed25519').textContent();

  // generate a name for the SSH key
  const sshKeyName = `test-${Math.random().toString(36).substring(2, 15)}`;

  // add the SSH key to GitHub
  const response = await fetch('https://api.github.com/user/keys', {
    method: 'POST',
    headers: {
      'Authorization': `token ${githubToken}`,
      'Content-Type': 'application/json',
      'Accept': 'application/vnd.github.v3+json'
    },
    body: JSON.stringify({ title: sshKeyName, key: key })
  });

  if (response.status !== 201) {
    throw new Error(`Failed to add SSH key to GitHub. Status: ${response.status}`);
  }

  // get the key ID from the response
  const data = await response.json();
  const keyId = data.id;
  // at the end of the test, or on failure, reset the GitHub repo
  test.afterAll(async () => {
    await resetGithub(githubToken, gitopsOwner, gitopsRepo, keyId, randomBranch);
  });

  // enable gitops now that the key is added
  await page.getByText('Test connection to repository').click();
  await expect(page.getByText('GitOps is enabled')).toBeVisible();




  await page.getByText('DoesNotExist')

  
});

async function trivialConfig(page: Page) {
  await page.getByRole('link', { name: 'Config', exact: true }).click();
  await page.getByLabel('Trivial Config').check();
  await page.getByRole('button', { name: 'Save config' }).click();
  await page.getByRole('button', { name: 'Go to updated version' }).click();
}

async function resetGithub(ghToken: string, owner: string, repo: string, keyId: string, branch: string) {
  // delete the key from GitHub
  await fetch(`https://api.github.com/repos/${owner}/${repo}/keys/${keyId}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `token ${ghToken}`,
      'Accept': 'application/vnd.github.v3+json'
    }
  });

  // delete the branch from GitHub
  await fetch(`https://api.github.com/repos/${owner}/${repo}/git/refs/heads/${branch}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `token ${ghToken}`,
      'Accept': 'application/vnd.github.v3+json'
    }
  });
}
