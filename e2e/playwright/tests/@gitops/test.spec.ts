import { test, expect, Page } from '@playwright/test';
import { login, uploadLicense } from '../shared';

test('gitops install', async ({ page }) => {
  test.setTimeout(120000); // 2 minutes
  // configure gitops
  const gitopsOwner = 'replicated-testim-kotsadm-gitops';
  const gitopsRepo = 'qakots-kotsadm-gitops';
  const testAppSlug = 'gitops-bobcat';
  const githubToken = process.env.GITOPS_GITHUB_TOKEN;
  if (!githubToken) {
    throw new Error('GITOPS_GITHUB_TOKEN is not set');
  }

  await login(page);
  await uploadLicense(page, expect, "gitops.yaml");
  await expect(page.locator('#app')).toContainText('Application Configuration', { timeout: 15000 });
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#app')).toContainText('Currently deployed version', { timeout: 15000 });
  await expect(page.locator('#app')).toContainText('Ready', { timeout: 30000 });

  // the app is now installed and ready, and the real test can begin

  // create a new version with a config change
  await trivialConfig(page, false);

  // generate a random branch name to use for this test
  const randomBranch = `test-${Math.random().toString(36).substring(2, 15)}`;

  // generate a random path to use for this test
  const randomPath = `/test-${Math.random().toString(36).substring(2, 15)}`;

  await page.getByText('GitOps').click();
  await expect(page.getByText('GitHub')).toBeVisible();
  await filloutGitopsForm(page, gitopsOwner, gitopsRepo, randomBranch, randomPath);
  await page.getByRole('button', { name: 'Generate SSH key' }).click();
  await expect(page.getByText('ssh-ed25519')).toBeVisible();
  await expect(page.getByText('Copy key')).toBeVisible();
  // get the key text
  const key = await page.getByText('ssh-ed25519').textContent();

  // generate a name for the SSH key
  const sshKeyName = `test-${Math.random().toString(36).substring(2, 15)}`;

  // add the SSH key to GitHub
  const response = await fetch(`https://api.github.com/repos/${gitopsOwner}/${gitopsRepo}/keys`, {
    method: 'POST',
    headers: {
      'Authorization': `token ${githubToken}`,
      'Content-Type': 'application/json',
      'Accept': 'application/vnd.github.v3+json'
    },
    body: JSON.stringify({ title: sshKeyName, key: key })
  });
  console.log("key name", sshKeyName);

  if (response.status !== 201) {
    throw new Error(`Failed to add SSH key to GitHub. Status: ${response.status} Contents: ${await response.text()}`);
  }

  // get the key ID from the response
  const data = await response.json();
  const keyId = data.id;
  // at the end of the test, or on failure, reset the GitHub repo
  // TODO: figure out how to do this
  // test.afterAll(async () => {
  //   await resetGithub(githubToken, gitopsOwner, gitopsRepo, keyId, randomBranch);
  // });

  // enable gitops now that the key is added
  await page.getByRole('button', { name: 'Test connection to repository' }).click();
  await expect(page.getByText('GitOps is enabled')).toBeVisible();
  await page.getByRole('button', { name: 'Go to dashboard' }).click();
  await page.getByRole("link", { name: "Version history" }).click();
  await expect(page.getByRole('button', { name: 'Deploy', exact: true })).not.toBeVisible();
  await expect(page.getByRole('button', { name: 'Redeploy', exact: true })).not.toBeVisible();
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
    throw new Error(`Failed to get commits from GitHub. Status: ${commitResponse.status} Contents: ${await commitResponse.text()}`);
  }
  const commitResponseJson = await commitResponse.json();
  // there should be 3 commits - one for the initial version, and then two for the two config changes
  checkCommitMessageExists(commitResponseJson, 'Updating GitOps to version 2')
  checkCommitMessageExists(commitResponseJson, 'Updating GitOps to version 1')
  checkCommitMessageExists(commitResponseJson, 'Updating GitOps to version 0')

  const contentResponse = await fetch(`https://api.github.com/repos/${gitopsOwner}/${gitopsRepo}/contents/${randomPath}/${testAppSlug}.yaml?ref=${randomBranch}`, {
    headers: {
      'Authorization': `token ${githubToken}`,
      'Accept': 'application/vnd.github.v3+json'
    }
  })
  if (contentResponse.status !== 200) {
    throw new Error(`Failed to get content from GitHub. Status: ${contentResponse.status} Contents: ${await contentResponse.text()}`);
  }
  const contentResponseJson = await contentResponse.json();
  checkFilePathExists(contentResponseJson, `${randomPath}/${testAppSlug}.yaml`)

  // turn off gitops so we can test the behavior of kots afterwards
  await resetGithub(githubToken, gitopsOwner, gitopsRepo, keyId, randomBranch);

  await page.locator('span').filter({ hasText: 'GitOps' }).click();
  await expect(page.getByTestId('gitops-enabled')).toBeVisible();
  await page.getByText('Disable GitOps for this app').click();
  await expect(page.getByText('Are you sure you want to disable GitOps for this application?')).toBeVisible();
  await page.getByRole('button', { name: 'Disable GitOps' }).click();
  await expect(page.getByText('Are you sure you want to disable GitOps for this application?')).not.toBeVisible();
  await expect(page.getByTestId('gitops-not-enabled')).toBeVisible();
  console.log('gitops disabled')

  // visit application page, ensure deploy and redeploy buttons are visible, deploy new version
  await page.getByText('Application', { exact: true }).click();
  await expect(page.getByRole('button', { name: 'Redeploy' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Deploy', exact: true })).toBeVisible();
  await page.waitForTimeout(1000);
  await page.getByRole('button', { name: 'Deploy', exact: true }).click();
  await expect(page.getByText('(Sequence ')).toBeVisible();
  await page.getByRole('button', { name: 'Yes, Deploy' }).click();
  await expect(page.getByText('Currently deployed version')).toBeVisible(); // ensure that a version is deployed
  await expect(page.locator('#app')).toContainText('Ready', { timeout: 30000 });
  console.log('new version deployed')

  // test reenabling gitops but with a failed ssh connection
  await page.locator('div').filter({ hasText: /^GitOps$/ }).click();
  await filloutGitopsForm(page, gitopsOwner, gitopsRepo, randomBranch, randomPath);
  await page.getByRole('button', { name: 'Generate SSH key' }).click();
  await page.getByRole('button', { name: 'Test connection to repository' }).click();
  await expect(page.getByText('Connection to repository failed')).toBeVisible(); // there is no key in the repo
  await page.getByRole('button', { name: 'Try again' }).click();
  await expect(page.getByText('Connection to repository failed')).toBeVisible();
  await page.getByRole('button', { name: 'Cancel' }).click();
  await page.getByRole('button', { name: 'Back to configuration' }).click();
  await expect(page.getByTestId('gitops-repository-access-needed')).toBeVisible(); // ensure that the 'there's no key in the repo' message is now visible
  await page.getByText('Application', { exact: true }).click();
  await expect(page.getByRole('button', { name: 'Redeploy' })).toBeVisible(); // ensure that we're still in non-gitops mode
  console.log('test complete')
});

async function trivialConfig(page: Page, isGitops: boolean) {
  await page.getByRole('list').getByRole('link', { name: 'Config' }).click();
  await expect(page.getByText('A trivial config item')).toBeVisible();
  if (await page.getByLabel('Trivial Config').isChecked()) {
    await page.getByLabel('Trivial Config').uncheck();
  } else {
    await page.getByLabel('Trivial Config').check();
  }
  await page.getByRole('button', { name: 'Save config' }).click();
  if (!isGitops) {
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
  console.log('github has been reset')
}

interface GitHubCommit {
  commit: {
    message: string;
  };
}

interface GitHubFile {
  path: string;
}

function checkCommitMessageExists(commits: GitHubCommit[], message: string) {
  const commit = commits.find((c) => c.commit.message === message);
  if (!commit) {
    throw new Error(`Commit message "${message}" not found in commits`);
  }
}

function checkFilePathExists(file: GitHubFile, path: string) {
  if ("/"+file.path !== path) {
    throw new Error(`File path "${path}" not found. Got "/${file.path}" instead`);
  }
}

async function filloutGitopsForm(page: Page, owner: string, repo: string, branch: string, path: string) {
  await expect(page.getByTestId('gitops-not-enabled')).toBeVisible();
  await expect(page.getByAltText('not_enabled')).toBeVisible();
  await page.waitForTimeout(1000);
  await page.getByPlaceholder('owner').click();
  await page.getByPlaceholder('owner').fill(owner);
  await page.getByPlaceholder('Repository').click();
  await page.getByPlaceholder('Repository').fill(repo);
  await page.getByPlaceholder('main').click();
  await page.getByPlaceholder('main').fill(branch);
  await page.getByPlaceholder('/path/to-deployment').click();
  await page.getByPlaceholder('/path/to-deployment').fill(path);

  // check if the owner field is actually filled out after sitting for 0.1 seconds
  await page.waitForTimeout(100);
  const ownerValue = await page.getByPlaceholder('owner').getAttribute('value');
  if (ownerValue === owner) {
    console.log('owner field is filled out')
  } else {
    throw new Error(`Owner field is not filled out. Got "${ownerValue}" instead of "${owner}"`);
  }
}
