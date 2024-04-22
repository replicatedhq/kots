import { test, expect } from '@playwright/test';
import { login, uploadLicense } from '../shared';

const { execSync } = require("child_process");

test('config validation', async ({ page }) => {
  await login(page);
  await uploadLicense(page, expect);
  await expect(page.locator('h3')).toContainText('Config Regex Group Validation', { timeout: 15000 });
  await page.getByRole('button', { name: 'Continue' }).click();
  await expect(page.locator('#email_text-errblock')).toContainText('This item is required');
  await page.locator('#email_text-group').getByRole('textbox').click();
  await page.locator('#email_text-group').getByRole('textbox').fill('test');
  await expect(page.locator('#email_text-group')).toContainText('A valid email address must be specified.');
  await expect(page.locator('#app')).toContainText('Error detected. Please use config nav to the left to locate and fix issues.');
  await page.locator('#email_text-group').getByRole('textbox').click();
  await page.locator('#email_text-group').getByRole('textbox').fill('test@email.com');
  await expect(page.getByText('A valid email address must be specified.')).not.toBeVisible();
  await expect(page.getByText('Error detected. Please use config nav to the left to locate and fix issues.')).not.toBeVisible();
  await page.locator('input[type="password"]').click();
  await page.locator('input[type="password"]').fill('dd');
  await expect(page.locator('#password-group')).toContainText('The password must be between 8 and 16 characters long and can contain a combination of uppercase letters, lowercase letters, digits, and special characters.');
  await expect(page.locator('#app')).toContainText('Error detected. Please use config nav to the left to locate and fix issues.');
  await page.locator('input[type="password"]').fill('password');
  await expect(page.getByText('The password must be between 8 and 16 characters long and can contain a combination of uppercase letters, lowercase letters, digits, and special characters.')).not.toBeVisible();
  await expect(page.getByText('Error detected. Please use config nav to the left to locate and fix issues.')).not.toBeVisible();
  await page.locator('textarea').click();
  await page.locator('textarea').fill('dd');
  await expect(page.locator('#cve_text_area-group')).toContainText('A valid CVE number must be in the format CVE-YYYY-NNNN, where YYYY is the year and NNNN is a number between 0001 and 9999.');
  await page.locator('textarea').fill('CVE-2023-1234');
  await expect(page.getByText('A valid CVE number must be in the format CVE-YYYY-NNNN, where YYYY is the year and NNNN is a number between 0001 and 9999.')).not.toBeVisible();
  await expect(page.getByText('Error detected. Please use config nav to the left to locate and fix issues.')).not.toBeVisible();
  await page.setInputFiles('input[type="file"]', `${process.env.TEST_PATH}/invalid-jwt.txt`);
  await expect(page.locator('#jwt_file-group')).toContainText('A valid JWT file must be in the format header.payload.signature.');
  await expect(page.locator('#app')).toContainText('Error detected. Please use config nav to the left to locate and fix issues.');
  await page.setInputFiles('input[type="file"]', `${process.env.TEST_PATH}/valid-jwt.txt`);
  await expect(page.getByText('A valid JWT file must be in the format header.payload.signature.')).not.toBeVisible();
  await expect(page.getByText('Error detected. Please use config nav to the left to locate and fix issues.')).not.toBeVisible();
  await page.getByLabel('Customize domain name').check();
  await page.locator('#domain_name-group').getByRole('textbox').click();
  await page.locator('#domain_name-group').getByRole('textbox').fill('okay.domain.com');
  await page.getByRole('button', { name: 'Continue' }).click();
  await page.getByRole('link', { name: 'Config', exact: true }).click();
  await page.locator('#email_text-group').getByRole('textbox').click();
  await page.locator('#email_text-group').getByRole('textbox').fill('');
  await page.getByRole('button', { name: 'Save config' }).click();
  await expect(page.locator('#email_text-errblock')).toContainText('This item is required');
  await page.locator('#email_text-group').getByRole('textbox').click();
  await page.locator('#email_text-group').getByRole('textbox').fill('email.test@okay.com');
  await page.locator('input[type="password"]').click();
  await page.locator('input[type="password"]').fill('dd');
  await expect(page.locator('#password-group')).toContainText('The password must be between 8 and 16 characters long and can contain a combination of uppercase letters, lowercase letters, digits, and special characters.');
  await expect(page.locator('#app')).toContainText('Error detected. Please use config nav to the left to locate and fix issues.');
  await page.locator('input[type="password"]').fill('password');
  await expect(page.getByText('The password must be between 8 and 16 characters long and can contain a combination of uppercase letters, lowercase letters, digits, and special characters.')).not.toBeVisible();
  await expect(page.getByText('Error detected. Please use config nav to the left to locate and fix issues.')).not.toBeVisible();
  await page.getByText('CVE-2023-').click();
  await page.getByText('CVE-2023-').fill('CVE-2023-123');
  await expect(page.locator('#cve_text_area-group')).toContainText('A valid CVE number must be in the format CVE-YYYY-NNNN, where YYYY is the year and NNNN is a number between 0001 and 9999.');
  await expect(page.locator('#app')).toContainText('Error detected. Please use config nav to the left to locate and fix issues.');
  await page.getByText('CVE-2023-').fill('CVE-2023-1234');
  await expect(page.getByText('A valid CVE number must be in the format CVE-YYYY-NNNN, where YYYY is the year and NNNN is a number between 0001 and 9999.')).not.toBeVisible();
  await expect(page.getByText('Error detected. Please use config nav to the left to locate and fix issues.')).not.toBeVisible();
  await page.setInputFiles('input[type="file"]', `${process.env.TEST_PATH}/invalid-jwt.txt`);
  await expect(page.locator('#jwt_file-group')).toContainText('A valid JWT file must be in the format header.payload.signature.');
  await expect(page.locator('#app')).toContainText('Error detected. Please use config nav to the left to locate and fix issues.');
  await page.setInputFiles('input[type="file"]', `${process.env.TEST_PATH}/valid-jwt.txt`);
  await expect(page.getByText('A valid JWT file must be in the format header.payload.signature.')).not.toBeVisible();
  await expect(page.getByText('Error detected. Please use config nav to the left to locate and fix issues.')).not.toBeVisible();
  await page.getByRole('button', { name: 'Save config' }).click();
  await expect(page.getByLabel('Next step').getByRole('paragraph')).toContainText('The config for Config Validation has been updated.', { timeout: 10000 });

  // validate the cli
  var invalidTestInput = [
    { key: "email_text", invalidValue: "invalid_email", error: "A valid email address must be specified." },
    { key: "cve_text_area", invalidValue: "CVE20221234", error: "A valid CVE number must be in the format CVEYYYYNNNN, where YYYY is the year and NNNN is a number between 0001 and 9999." },
    { key: "password", invalidValue: "short", error: "The password must be between 8 and 16 characters long and can contain a combination of uppercase letters, lowercase letters, digits, and special characters." },
  ];
  for (const i of invalidTestInput) {
    const setConfigEmailCmd = `kubectl kots set config ${process.env.APP_SLUG} -n=${process.env.NAMESPACE} --key=${i.key} --value=${i.invalidValue} --merge | grep -A1 "Errors:" | grep -v "Errors:"  | sed 's/^[[:space:]]*//;s/[-]*//g'`;
    const setConfigResult = execSync(setConfigEmailCmd).toString().trim();
    if (setConfigResult !== i.error) {
      throw new Error(`Expected error message "${i.error}" but got "${setConfigResult}"`);
    }
  }
});
