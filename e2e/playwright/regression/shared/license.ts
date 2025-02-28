import { Page, Expect } from '@playwright/test';

import { updateCustomer } from './vendor-api';

export const uploadLicense = async (page: Page, expect: Expect, licenseFile = "license.yaml") => {
  await page.setInputFiles('input[type="file"][accept="application/x-yaml,.yaml,.yml,.rli"]', `${process.env.TEST_PATH}/${licenseFile}`);
  await page.getByRole('button', { name: 'Upload license' }).click();
  await expect(page.locator('#app')).toContainText('Installing your license');
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

export async function validateUpdatedLicense(page: Page, expect: Expect, newIntEntitlement: number) {
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
  await expect(updateRow).toContainText('1.0.0');
  await expect(updateRow).toContainText('License Change');
  await expect(updateRow).toContainText('Sequence 2');
};