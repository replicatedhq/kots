import { test, expect, Page, Expect } from '@playwright/test';

import {
  generateSupportBundleUi,
  generateSupportBundleCli,
  validateRedaction,
  validateRedactionReport,
  validateDownloadBundle,
  validateGenerateBundleModal,
  validateSupportBundleDelete,
  configureURLRedaction,
} from '../../shared-core';

import {
  login,
  uploadLicense,
  appIsReady,
} from '../shared';

test('support bundle', async ({ page, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);

  // this seems to take a really long time on okd
  test.setTimeout(20 * 60 * 1000); // 20 minutes

  await login(page);
  await uploadLicense(page, expect);
  await appIsReady(page, expect);
  await configureURLRedaction(page, expect);
  await validateSupportBundleUi(page, expect);
  await validateSupportBundleCli(page, expect);
});

async function validateSupportBundleUi(page: Page, expect: Expect) {
  await generateSupportBundleUi(page, expect);
  await validateRedaction(page, expect, 2);
  await validateRedactionReport(page, expect);
  await validateDownloadBundle(page, expect, 2);
  await validateGenerateBundleModal(page, expect);
  await validateSupportBundleDelete(page, expect);
}

async function validateSupportBundleCli(page: Page, expect: Expect) {
  await generateSupportBundleCli(page, expect);
  await validateRedaction(page, expect, 2);
  await validateRedactionReport(page, expect);
  await validateDownloadBundle(page, expect, 2);
  await validateGenerateBundleModal(page, expect);
  await validateSupportBundleDelete(page, expect);
}
