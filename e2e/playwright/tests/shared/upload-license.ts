export const uploadLicense = async (page, expect) => {
  await page.setInputFiles('input[type="file"][accept="application/x-yaml,.yaml,.yml,.rli"]', `${process.env.TEST_PATH}/license.yaml`)
  await page.getByRole('button', { name: 'Upload license' }).click();
  await expect(page.locator('#app')).toContainText('Installing your license');
};
