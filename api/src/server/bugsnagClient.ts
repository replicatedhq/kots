import bugsnag from "@bugsnag/js";
import * as BugsnagCore from "@bugsnag/browser/dist/types/bugsnag-core";
let bugsnagClient;

export function createBugsnagClient(bugsnagOptions: BugsnagCore.IConfig): BugsnagCore.Client | void {
  if (bugsnagOptions.apiKey) {
    bugsnagClient = bugsnag({
      ...bugsnagOptions
    });
  }

  return bugsnagClient;
}

export function getBugsnagClient(): BugsnagCore.Client | void {
  return bugsnagClient;
}