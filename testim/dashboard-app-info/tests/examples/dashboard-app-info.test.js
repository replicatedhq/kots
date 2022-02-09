"use strict";

const { go, resize, evaluate, apiCall, waitForText, click, type, inputFile, exists, test, l, Locator } = require('testim');

Locator.set(require('./locators/locators.js'));

test("type=embedded cluster, env=online, phase=new install, rbac=cluster admin", async () => {
  await go("http://localhost:8800");
  await resize({width: 1440, height: 900});
  
  await evaluate(() => {
    exportsTest.testShouldResetPassword = true;
    exportsTest.testIsAirgapped = false;
  });
  
  async function qakots_login() {
    if(await evaluate(() => {
      try {
        return testShouldResetPassword;
      } catch (e) {
       return false;
      }
    })) {
      // Converting a 'cli-action-code-step' step has to be done manually at this time
    }
    await click(l("password"));
    await type(l("password"), 'password');
    await click(l("Log_in"));
    // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  }
  await qakots_login();
  async function qakots_dashboard_app_info() {
    await click(l("Application"));
    await click(l("Dashboard"));
    //TODO Please add an assertion here
    await exists(l("QAKots"));
    await waitForText(l("Ready"), 'Ready');
    //TODO Please add an assertion here
    if(await evaluate(() => {
      try {
        // Airgapped bundles don't have app link
      return !testIsAirgapped;
      } catch (e) {
       return false;
      }
    })) {
      //TODO Please add an assertion here
      await exists(l("Example_Application"));
    }
    //TODO Please add an assertion here
    await exists(l("1.0.0_Sequence_0_Currently_deployed"));
    //TODO Please add an assertion here
    await exists(l("Version_Check_for_update_Configure_"));
    //TODO Please add an assertion here
    await exists(l("License_Sync_license_type=embedded_"));
    //TODO Please add an assertion here
    await exists(l("Snapshots_Snapshot_settings_Start_s"));
    if(await evaluate(() => {
      try {
        return !testIsAirgapped;
      } catch (e) {
       return false;
      }
    })) {
      //TODO Please add an assertion here
      await exists(l("Check_for_update"));
    }
    if(await evaluate(() => {
      try {
        return testIsAirgapped;
      } catch (e) {
       return false;
      }
    })) {
      //TODO Please add an assertion here
      await exists(l("Upload_new_version"));
    }
    //TODO Please add an assertion here
    await exists(l("Configure_automatic_updates"));
    if(await evaluate(() => {
      try {
        return testIsVeleroInstalled;
      } catch (e) {
       return false;
      }
    })) {
      //TODO Please add an assertion here
      await exists(l("Snapshot_settings"));
    }
    if(await evaluate(() => {
      try {
        return testIsVeleroInstalled;
      } catch (e) {
       return false;
      }
    })) {
      //TODO Please add an assertion here
      await exists(l("Start_snapshot"));
    }
    //TODO Please add an assertion here
    await exists(l("Sync_license"));
    //TODO Please add an assertion here
    await exists(l("See_all_versions"));
    if(await evaluate(() => {
      try {
        return testIsVeleroInstalled;
      } catch (e) {
       return false;
      }
    })) {
      //TODO Please add an assertion here
      await exists(l("See_all_snapshots"));
    }
    //TODO Please add an assertion here
    await exists(l("See_license_details"));
    await waitForText(l("Ready"), 'Ready');
  }
  await qakots_dashboard_app_info();

}); // end of test
