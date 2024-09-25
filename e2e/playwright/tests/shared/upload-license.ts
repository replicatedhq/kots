export const uploadLicense = async (page, expect, licenseFile = "license.yaml") => {
  await page.setInputFiles('input[type="file"][accept="application/x-yaml,.yaml,.yml,.rli"]', `${process.env.TEST_PATH}/${licenseFile}`);
  await page.getByRole('button', { name: 'Upload license' }).click();
  await expect(page.locator('#app')).toContainText('Installing your license');
};
