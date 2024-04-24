export const validateDeployLogs = async (page, expect) => {
  await expect(page.getByText('dryrunStdout')).toBeVisible();
  await expect(page.getByText('dryrunStderr')).toBeVisible();
  await expect(page.getByText('applyStdout')).toBeVisible();
  await expect(page.getByText('applyStderr')).toBeVisible();
  await expect(page.getByText('helmStdout')).toBeVisible();
  await expect(page.getByText('helmStderr')).toBeVisible();
  await page.getByText('dryrunStderr').click();
  await page.getByText('applyStdout').click();
  await expect(page.locator('.view-lines')).toContainText('created');
  await page.getByRole('button', { name: 'Ok, got it!' }).click();
};
