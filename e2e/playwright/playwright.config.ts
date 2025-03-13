import { defineConfig, devices } from '@playwright/test';

/**
 * Read environment variables from file.
 * https://github.com/motdotla/dotenv
 */
// require('dotenv').config();

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: process.env.TEST_DIR || './tests',
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['line'],
    ['html', { open: 'never' }],
  ],
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: process.env.BASE_URL || `http://localhost:${process.env.PORT || 8800}`,

    /*
      To include traces for failed tests, set this to 'retain-on-failure'.
      This is not enabled by default because it's performance heavy.
      See https://playwright.dev/docs/trace-viewer.
    */
    trace: 'retain-on-failure',

    /* Screenshot on failure. */
    screenshot: 'only-on-failure',

    /* Timeout for each action in milliseconds. Defaults to 0 (no limit). */
    actionTimeout: 15 * 1000,

    /* Timeout for each navigation in milliseconds. Defaults to 0 (no limit). */
    navigationTimeout: 30 * 1000,
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
