"use strict";

const { Builder, Key, until } = require('selenium-webdriver');

(async () => {
  const driver = await new Builder().forBrowser('chrome').build();
  try {
  await driver.get("http://localhost:8800");
  await Login_and_Upload_License();
  await Analyze_app();
  await Generate_and_Validate_Bundle_CLI();
  } finally {
    await driver.quit();
  }
})();

// move to utils.js

async function click(driver, selector, button = 0) {
  const element = await driver.findElement({css: selector});
  await driver.actions()
    .click(element, button)
    .perform();
}

async function getText(driver, selector) {
  const element = await driver.findElement({css: selector});
  return await element.getText();
}

async function isVisible(driver, selector) {
  const element = await driver.findElement({css: selector});
  return element.isDisplayed();
}
async function sendKeys(driver, selector, keys) {
  const element = await driver.findElement({css: selector});
  await element.sendKeys(keys);
}

async function waitForElementVisible(driver, selector, timeout = 30000) {
  await driver.wait(until.elementIsVisible(await driver.findElement({css: selector})), timeout);
}

async function waitForText(driver, selector, expectedText, timeout = 30000) {
  await driver.wait(() => {
    return driver.executeScript((selector, expectedText) => {
      const element = document.querySelector(selector);
      return element && element.textContent.replace(/[\r\n]+/g, "").trim() === expectedText.trim();
    }, selector, expectedText);
  }, timeout);
}


// shared steps \\
async function Login_and_Upload_License() {
  await click(driver, "[type='password']");
  await sendKeys(driver, "[type='password']", 'password');
  await click(driver, "[type='submit']");
  await waitForText(driver, ".u-fontSize--header", 'Upload your license file');
  await waitForElementVisible(driver, "[class^='u-marginTop'], [class*=' u-marginTop'] .u-textAlign--center");
  // Converting a 'drop-file' step has to be done manually at this time
  await click(driver, "[type='button']");
  //TODO Please add an assertion here
await isVisible(driver, "[class^='u-paddingRight'], [class*=' u-paddingRight'] .dashboard-card");
  await waitForText(driver, "[class^='u-marginLeft'], [class*=' u-marginLeft'] .u-fontSize--normal", 'Ready');
}

async function qakots_delete_support_bundle() {
  await click(driver, ".centered-container .u-overflow--auto > :nth-child(1) [class^='tw-ml'], [class*=' tw-ml']");
  await waitForElementVisible(driver, ".tw-absolute");
  await click(driver, ".tw-underline");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  //TODO Please add an assertion here
await isVisible(driver, ".centered-container .u-overflow--auto > :nth-child(1) .bundle-row-wrapper");
  await click(driver, ".centered-container .u-overflow--auto > div:nth-of-type(1) [class^='tw-ml'], [class*=' tw-ml']");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
}

async function Analyze_app() {
  await click(driver, ".left-items > div:nth-of-type(1) .flex");
  await click(driver, "ul > li:nth-of-type(4) a");
  await click(driver, ".u-fontSize--small");
  await waitForElementVisible(driver, ".Modal-body");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  await click(driver, ".action-tab-bar > :nth-child(1)");
  await click(driver, "[type='text']");
  await sendKeys(driver, "[type='text']", 'https://raw.githubusercontent.com/replicatedhq/kots/master/testim/testim-redactor-spec.yaml');
  await click(driver, "body > :nth-child(11) .primary");
  await click(driver, ".u-marginRight--10");
  await click(driver, ".primary");
  await waitForElementVisible(driver, ".SupportBundleDetails--Progress > :nth-child(1)");
  //TODO Please add an assertion here
await isVisible(driver, ".SupportBundleDetails--Progress");
  await waitForElementVisible(driver, "[class^='u-marginTop'], [class*=' u-marginTop'] .is-active");
  await driver.sleep(10000);
  //TODO Please add an assertion here
await isVisible(driver, ".action-content");
  await click(driver, ".tab-items > :nth-child(2)");
  await click(driver, "[for='sub-dir-cluster-info-1-cluster-info-3-0']");
  await click(driver, "[title='cluster_version.json'] div");
  //TODO Please add an assertion here
await getText(driver, ".ace_content");
  // Converting a 'validation-code-step' step has to be done manually at this time
  await click(driver, ".tab-items > :nth-child(3)");
  await waitForText(driver, ".action-content > :nth-child(1) > :nth-child(2) .u-fontSize--large", 'IP Addresses.regex.0');
  //TODO Please add an assertion here
await isVisible(driver, ".action-content > :nth-child(1) > :nth-child(2)");
  await click(driver, ".action-content > :nth-child(1) > :nth-child(2) .replicated-link");
  await click(driver, ".Timeline--wrapper .u-color--dustyGray");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  //TODO Please add an assertion here
await isVisible(driver, ".AceEditor");
  //TODO Please add an assertion here
await isVisible(driver, ".redactor-pager .flex");
  //TODO Please add an assertion here
await isVisible(driver, ".primary");
  await click(driver, ".btn");
  // Converting a 'cli-validation-download-file' step has to be done manually at this time
  await click(driver, ".link");
  await waitForElementVisible(driver, ".centered-container .u-overflow--auto > :nth-child(1) .bundle-row-wrapper");
  //TODO Please add an assertion here
await isVisible(driver, ".SupportBundleRow--Progress > :nth-child(1) .clickable");
  await click(driver, ".centered-container a");
  await waitForElementVisible(driver, "[type='button']");
  await click(driver, "body > :nth-child(11) a");
  //TODO Please add an assertion here
await isVisible(driver, ".react-prism");
  //TODO Please add an assertion here
await isVisible(driver, "body > :nth-child(11) .u-linkColor");
  await click(driver, "[class^='u-padding'], [class*=' u-padding'] > div:nth-of-type(4) .btn");
  await qakots_delete_support_bundle();
}

async function Generate_and_Validate_Bundle_CLI() {
  if (!(await isVisible(driver, "ul > li:nth-of-type(4) a"))) {
      await click(driver, ".NavBarWrapper .is-active .text span");
    }
  await click(driver, "ul > li:nth-of-type(4) a");
  await waitForElementVisible(driver, ".centered-container a");
  if (await getText(driver, ".WatchDetailPage--wrapper .btn") === "'Generate a support bundle'") {
      await click(driver, ".centered-container a");
    }
  await waitForElementVisible(driver, ".u-fontSize--larger");
  await click(driver, "body > :nth-child(11) a");
  await waitForElementVisible(driver, ".react-prism");
  //TODO Please add an assertion here
await getText(driver, ".react-prism");
  // Converting a 'cli-action-code-step' step has to be done manually at this time
  await waitForElementVisible(driver, "[class^='u-marginTop'], [class*=' u-marginTop'] .is-active");
  await driver.sleep(10000);
  //TODO Please add an assertion here
await isVisible(driver, ".action-content");
  await click(driver, ".tab-items > :nth-child(2)");
  await click(driver, "[for='sub-dir-cluster-info-1-cluster-info-3-0']");
  await click(driver, "[title='cluster_version.json'] div");
  //TODO Please add an assertion here
await getText(driver, ".ace_content");
  // Converting a 'validation-code-step' step has to be done manually at this time
  await click(driver, ".tab-items > :nth-child(3)");
  await waitForText(driver, ".action-content > :nth-child(1) > :nth-child(2) .u-fontSize--large", 'IP Addresses.regex.0');
  //TODO Please add an assertion here
await isVisible(driver, ".action-content > :nth-child(1) > :nth-child(2)");
  await click(driver, ".action-content > :nth-child(1) > :nth-child(2) .replicated-link");
  await click(driver, ".Timeline--wrapper .u-color--dustyGray");
  //TODO Please add an assertion here
await isVisible(driver, ".AceEditor");
  //TODO Please add an assertion here
await isVisible(driver, ".redactor-pager .flex");
  //TODO Please add an assertion here
await isVisible(driver, ".primary");
  await click(driver, ".btn");
  // Converting a 'cli-validation-download-file' step has to be done manually at this time
  await click(driver, ".link");
  await waitForElementVisible(driver, ".centered-container .u-overflow--auto > :nth-child(1) .bundle-row-wrapper");
  //TODO Please add an assertion here
await isVisible(driver, ".SupportBundleRow--Progress > :nth-child(1) .clickable");
  await click(driver, ".centered-container a");
  await waitForElementVisible(driver, "[type='button']");
  await click(driver, "body > :nth-child(11) a");
  //TODO Please add an assertion here
await isVisible(driver, ".react-prism");
  //TODO Please add an assertion here
await isVisible(driver, "body > :nth-child(11) .u-linkColor");
  await click(driver, "[class^='u-padding'], [class*=' u-padding'] > div:nth-of-type(4) .btn");
  await qakots_delete_support_bundle();
}

