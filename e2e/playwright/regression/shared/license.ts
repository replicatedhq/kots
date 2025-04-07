import { Page, Expect } from '@playwright/test';

import { updateCustomer } from './api';

export const uploadLicense = async (page: Page, expect: Expect, licenseFile: string = "license.yaml") => {
  await page.setInputFiles('input[type="file"]', `${process.env.TEST_PATH}/${licenseFile}`);
  await page.getByRole('button', { name: 'Upload license' }).click();
  await expect(page.locator('#app')).toContainText('Installing your license');
};

export const validateDuplicateLicenseUpload = async (page: Page, expect: Expect) => {
  const navbarDropdownContainer = page.getByTestId("navbar-dropdown-container");
  await expect(navbarDropdownContainer).toBeVisible();
  await navbarDropdownContainer.getByTestId("navbar-dropdown-button").click();
  await navbarDropdownContainer.getByTestId("add-new-application").click();

  await uploadLicense(page, expect);

  const uploadLicenseError = page.getByTestId("upload-license-error");
  await expect(uploadLicenseError).toBeVisible({ timeout: 15000 });
  await uploadLicenseError.getByTestId("view-more-button").click();

  const uploadLicenseErrorModal = page.getByTestId("upload-license-error-modal");
  await expect(uploadLicenseErrorModal).toBeVisible();
  await expect(uploadLicenseErrorModal).toContainText("License already exist");
  await expect(uploadLicenseErrorModal.getByTestId("remove-app-instructions")).toBeVisible();
  await uploadLicenseErrorModal.getByRole('button', { name: 'Ok, got it!' }).click();
  await expect(uploadLicenseErrorModal).not.toBeVisible();
};

export const validateCurrentLicense = async (page: Page, expect: Expect, customerName: string, channelName: string, isAirgapSupported: boolean, isEC: boolean) => {
  await page.getByRole('link', { name: 'License', exact: true }).click();

  const licenseCard = page.getByTestId('license-card');
  await expect(licenseCard).toBeVisible({ timeout: 15000 });
  await expect(licenseCard.getByTestId('license-customer-name')).toHaveText(customerName);
  await expect(licenseCard.getByTestId('license-channel-name')).toHaveText(channelName);
  await expect(licenseCard.getByTestId('license-type')).toHaveText('Prod license');
  await expect(licenseCard.getByTestId('license-expiration-date')).toHaveText('Does not expire');
  
  if (isAirgapSupported) {
    await expect(licenseCard).toContainText('Airgap enabled');
  }

  if (isEC) {
    await expect(licenseCard).toContainText('Disaster Recovery enabled');
  } else {
    await expect(licenseCard).toContainText('Snapshots enabled');
  }

  await expect(licenseCard).toContainText('GitOps enabled');
};

export async function updateOnlineLicense(page: Page, customerId: string, customerName: string, channelId: string, isAirgapSupported: boolean, isEC: boolean): Promise<number> {
  const newIntEntitlement = Math.floor(Math.random() * 1000);
  await updateCustomer(customerId, customerName, channelId, isAirgapSupported, isEC, newIntEntitlement);
  await page.getByRole('button', { name: 'Sync license' }).click();
  return newIntEntitlement;
}

export async function validateUpdatedLicense(page: Page, expect: Expect, newIntEntitlement: number, expectedSequence: number) {
  const nextStepModal = page.getByTestId("license-next-step-modal");
  await expect(nextStepModal).toBeVisible({ timeout: 30000 });
  await nextStepModal.getByRole('button', { name: 'Cancel' }).click();
  await expect(nextStepModal).not.toBeVisible();

  const licenseCard = page.getByTestId('license-card');
  await licenseCard.getByTestId("view-license-entitlements-button").click();
  const entitlements = page.getByTestId("license-entitlements");
  await expect(entitlements).toBeVisible();
  await expect(entitlements).toContainText(newIntEntitlement.toString());

  await page.getByRole('link', { name: 'Version history', exact: true }).click();
  const updatesCard = page.getByTestId('available-updates-card');
  await expect(updatesCard).toBeVisible();
  const updateRow = updatesCard.getByTestId('version-history-row-0');
  await expect(updateRow).toBeVisible();
  await expect(updateRow).toContainText('License Change');
  await expect(updateRow).toContainText(`Sequence ${expectedSequence}`);
};

export async function updateAirgappedLicense(page: Page, expect: Expect, newLicenseFile: string) {
  const licenseCard = page.getByTestId('license-card');
  await expect(licenseCard).toBeVisible({ timeout: 15000 });
  await page.setInputFiles('[data-testid="license-upload-dropzone"] input', `${process.env.TEST_PATH}/${newLicenseFile}`);
};
