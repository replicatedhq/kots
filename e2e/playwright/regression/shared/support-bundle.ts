import { Page, Expect } from '@playwright/test';

import {
  setRedactSpec,
  configureURLRedaction,
  generateSupportBundleUi,
  validateRedaction,
  validateRedactionReport,
  validateDownloadBundle,
} from '../../shared-core';

export async function validateGenerateSupportBundleUi(page: Page, expect: Expect, isAirgapped: boolean) {
  if (isAirgapped) {
    await setRedactSpec(page, expect);
  } else {
    await configureURLRedaction(page, expect);
  }
  await generateSupportBundleUi(page, expect);
  await validateRedaction(page, expect, 4);
  await validateRedactionReport(page, expect);
  await validateDownloadBundle(page, expect, 4);
}
