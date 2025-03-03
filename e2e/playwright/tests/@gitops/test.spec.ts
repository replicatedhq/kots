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
  // TODO: reenable outside of local dev
  // await expect(page.locator('#app')).toContainText('Ready', { timeout: 30000 });

  // the app is now installed and ready, and the real test can begin

  // create a new version with a config change
  await trivialConfig(page, false);

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
  await page.getByText('Go to dashboard').click();
  await page.getByText('Version history').click();
  await expect(page.getByText('Deploy')).not.toBeVisible();
  await expect(page.getByText('Redeploy')).not.toBeVisible();
  // wait 10 seconds for things to actually be committed
  await page.waitForTimeout(10000);
  await trivialConfig(page, true);
  

  const commitResponse = await fetch(`https://api.github.com/repos/${gitopsOwner}/${gitopsRepo}/commits?sha=${randomBranch}`, {
    headers: {
      'Authorization': `token ${githubToken}`,
      'Accept': 'application/vnd.github.v3+json'
    }
  })
  if (commitResponse.status !== 200) {
    throw new Error(`Failed to get commits from GitHub. Status: ${commitResponse.status}`);
  }
  const commits = await commitResponse.json();
  console.log(commits);
  // TODO validate that appropriate commits are present

  const contentResponse = await fetch(`https://api.github.com/repos/${gitopsOwner}/${gitopsRepo}/contents/${randomPath}/${testAppSlug}.yaml?ref=${randomBranch}`, {
    headers: {
      'Authorization': `token ${githubToken}`,
      'Accept': 'application/vnd.github.v3+json'
    }
  })
  if (contentResponse.status !== 200) {
    throw new Error(`Failed to get content from GitHub. Status: ${contentResponse.status}`);
  }
  const content = await contentResponse.json();
  console.log(content);
  // TODO validate that the content is correct

  // turn off gitops so we can test the behavior of kots afterwards
  await resetGithub(githubToken, gitopsOwner, gitopsRepo, keyId, randomBranch);

  await page.getByText('GitOps').click();
  await page.getByText('Disable GitOps for this app').click();
  await expect(page.getByText('Are you sure you want to disable GitOps for this application?')).toBeVisible();
  await page.getByText('Disable GitOps').click();
  await expect(page.getByText('Are you sure you want to disable GitOps for this application?')).not.toBeVisible();

  // TODO: visit application page, ensure deploy and redeploy buttons are visible, deploy new version

  // TODO: test reenabling gitops but with a failed ssh connection

  await page.getByText('DoesNotExist')  
});

async function trivialConfig(page: Page, isGitops: boolean) {
  await page.getByRole('link', { name: 'Config', exact: true }).click();
  await page.getByLabel('Trivial Config').check();
  await page.getByRole('button', { name: 'Save config' }).click();
  if (isGitops) {
    await page.getByRole('button', { name: 'Go to updated version' }).click();
  } else {
    await page.getByRole('button', { name: 'Ok, got it!' }).click();
    await page.waitForTimeout(3000);
  }
}


let hasResetGithub = false; 
async function resetGithub(ghToken: string, owner: string, repo: string, keyId: string, branch: string) {
  if (hasResetGithub) {
    return;
  }
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
  hasResetGithub = true;
}
