import { test, expect } from '@playwright/test';
import { login, uploadLicense } from '../shared';

const { execSync } = require("child_process");

test('backup and restore', async ({ page }) => {
  await login(page);
  await uploadLicense(page, expect);
  
});
