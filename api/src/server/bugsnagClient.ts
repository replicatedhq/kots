import bugsnag from "@bugsnag/js";
import * as BugsnagCore from "@bugsnag/browser/dist/types/bugsnag-core";
let bugsnagClient;

/**
 * Creates a brand new Bugsnag client and stores it in the global namespace of bugsnagClient.ts.
 * If no options.apiKey is sent in, this method will return `undefined`.
 * @param bugsnagOptions {Object} - Options for Bugsnag
 * @return {undefined|BugsnagCore.Client}
 */
export function createBugsnagClient(bugsnagOptions: BugsnagCore.IConfig): BugsnagCore.Client | void {
  if (bugsnagOptions.apiKey) {
    bugsnagClient = bugsnag({
      ...bugsnagOptions
    });
  }

  return bugsnagClient;
}

/**
 * Grabs a previously initialized BugsnagClient. If the client is unavailable, this method returns `undefined`
 * @return {undefined|BugsnagCore.Client}
 */
export function getBugsnagClient(): BugsnagCore.Client | void {
  return bugsnagClient;
}