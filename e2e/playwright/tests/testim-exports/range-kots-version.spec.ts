"use strict";

const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({headless: false});
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto("http://localhost:30880");
  await export_test_vars();
  await page.click("[type='password']");
  await page.type("[type='password']", 'password');
  await page.click("[type='submit']");
  await Test_Airgap();
  await page.click(".navbar-dropdown-container span");
  await page.click(".dropdown-nav-menu > li:nth-of-type(2) p");
  // Converting a 'cli-action-code-step' step has to be done manually at this time
  // Converting a 'cli-action-code-step' step has to be done manually at this time
  await page.waitFor(30000);
  await page.reload();
  await Test_Online();
  await browser.close();
})();

// move to utils.js

async function waitForText(page, selector, expectedText) {
  await page.waitForFunction(([selector, expectedText]) => {
    const element = document.querySelector(selector);
    return element && element.textContent.replace(/[\r\n]+/g, "").trim() === expectedText.trim();
  }, [selector, expectedText]);
}

async function scrollOnElement(page, selector, x, y) {
  await page.evaluate(([selector, x, y]) => {
    const element = document.querySelector(selector);
    element.scroll(x, y);
  }, [selector, x, y]);
}

async function isVisible(page, selector) {
  try {
    const elementHandle = await page.$(selector);
    if(!elementHandle) {
      return false;
    }
    const elementBox = await elementHandle.boundingBox();
    return Boolean(elementBox);
  } catch(err) {
    return false;
  }
}
function getText(page, selector) {
  return page.evaluate((selector) => {
    const element = document.querySelector(selector);
    return element.textContent;
  }, selector);
}


// shared steps \\
async function export_test_vars() {
  await page.evaluate(() => {
    exportsTest.testNamespace = "range-kots-version";
    exportsTest.testAppSlug = "range-kots-version";
    exportsTest.testVendorAppID = "24Z2zibdnOgL5LfhQtEBdMolJlq";
    exportsTest.testChannelID = "24Z39f0sNqo48VV9ZpLIUtlhHrN";
    exportsTest.testVendorRestrictiveReleaseSemver = "v1.0.1";
    exportsTest.testVendorPermissiveReleaseSemver = "v1.0.0";
    exportsTest.testRestrictiveTargetKotsVersion = "11000.0.0";
    exportsTest.testPermissiveTargetKotsVersion = "10000.0.0";
    exportsTest.testReplicatedApiToken = "TODO_ADD_REPLICATED_API_TOKEN_FROM_SECRET";
    
  });
}

async function validate_error_message(errorMsg) {
  await page.evaluate((errorMsg) => {
    return errorMsg.includes("requires") && errorMsg.includes(testRestrictiveTargetKotsVersion);
  }, errorMsg);
}

async function validate_error_message_1(errorMsg) {
  await page.evaluate((errorMsg) => {
    return errorMsg.includes("requires") && errorMsg.includes(testRestrictiveTargetKotsVersion);
  }, errorMsg);
}

async function Test_Airgap() {
  // Converting a 'input-file' step has to be done manually at this time
  await page.click("[type='button']");
  await page.click("[placeholder='artifactory.some-big-bank.com']");
  await page.type("[placeholder='artifactory.some-big-bank.com']", 'ttl.sh');
  await page.click("[placeholder='username']");
  await page.type("[placeholder='username']", 'admin');
  await page.click("[type='password']");
  await page.type("[type='password']", 'admin');
  await page.click("[placeholder='namespace']");
  await page.type("[placeholder='namespace']", 'test');
  // Converting a 'input-file' step has to be done manually at this time
  await page.click("[type='button']");
  await page.waitFor(2000);
  await scrollOnElement(page, ".UploadLicenseFile--wrapper", 0, 500);
  //TODO Please add an assertion here
await getText(page, ".u-textColor--error");
  await validate_error_message(errorMsg);
  await page.click("[placeholder='artifactory.some-big-bank.com']");
  await page.type("[placeholder='artifactory.some-big-bank.com']", 'ttl.sh');
  await page.click("[placeholder='username']");
  await page.type("[placeholder='username']", 'admin');
  await page.click("[type='password']");
  await page.type("[type='password']", 'admin');
  await page.click("[placeholder='namespace']");
  await page.type("[placeholder='namespace']", 'test');
  await scrollOnElement(page, ".UploadLicenseFile--wrapper", 0, 600);
  await page.click(".LoginBox-wrapper .replicated-link");
  // Converting a 'input-file' step has to be done manually at this time
  await page.click("[type='button']");
  await waitForText(page, ".u-fontSize--larger", 'Uploading your airgap bundle');
  await waitForText(page, "[class^='u-marginLeft'], [class*=' u-marginLeft'] .u-fontSize--normal", 'Ready');
  await page.click("ul > li:nth-of-type(2) a");
  await page.waitFor(2000);
  await waitForText(page, ".Footer-wrapper .u-fontSize--small", `v${testPermissiveTargetKotsVersion} available.`);
  // Converting a 'input-file' step has to be done manually at this time
  await page.waitFor(2000);
  //TODO Please add an assertion here
await getText(page, ".u-textColor--error");
  await validate_error_message_1(errorMsg);
}

async function Get_release_sequence(desiredSemver) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function Promote(sequenceToPromote, versionLabel) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function Promote_restrictive_vendor_release() {
  await Get_release_sequence(testVendorRestrictiveReleaseSemver);
  await Promote(sequenceToPromote, testVendorRestrictiveReleaseSemver);
}

async function validate_error_message_2(errorMsg) {
  await page.evaluate((errorMsg) => {
    return errorMsg.includes("requires") && errorMsg.includes(testRestrictiveTargetKotsVersion);
  }, errorMsg);
}

async function Get_release_sequence_1(desiredSemver) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function Promote_1(sequenceToPromote, versionLabel) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function Promote_permissive_vendor_release() {
  await Get_release_sequence_1(testVendorPermissiveReleaseSemver);
  await Promote_1(sequenceToPromote, testVendorPermissiveReleaseSemver);
}

async function Get_release_sequence_2(desiredSemver) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function Promote_2(sequenceToPromote, versionLabel) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function Promote_restrictive_vendor_release_1() {
  await Get_release_sequence_2(testVendorRestrictiveReleaseSemver);
  await Promote_2(sequenceToPromote, testVendorRestrictiveReleaseSemver);
}

async function validate_error_message_3(errorMsg) {
  await page.evaluate((errorMsg) => {
    return errorMsg.includes("requires") && errorMsg.includes(testRestrictiveTargetKotsVersion);
  }, errorMsg);
}

async function validate_error_message_4(errorMsg) {
  await page.evaluate((errorMsg) => {
    return errorMsg.includes("requires") && errorMsg.includes(testRestrictiveTargetKotsVersion);
  }, errorMsg);
}

async function validate_view_files_seqience() {
  await page.evaluate(() => {
    return window.location.pathname.endsWith("/0");
  });
}

async function Test_Online() {
  await Promote_restrictive_vendor_release();
  // Converting a 'input-file' step has to be done manually at this time
  await page.click("[type='button']");
  await waitForText(page, "[class^='u-paddingTop'], [class*=' u-paddingTop']", 'Install in airgapped environment');
  await scrollOnElement(page, ".UploadLicenseFile--wrapper", 0, 600);
  await page.click(".UploadLicenseFile--wrapper > div:nth-of-type(2) .link");
  await waitForText(page, "[class^='u-paddingTop'], [class*=' u-paddingTop']", 'Installing your license');
  await waitForText(page, "[class^='u-paddingTop'], [class*=' u-paddingTop']", 'Install in airgapped environment');
  await scrollOnElement(page, ".UploadLicenseFile--wrapper", 0, 600);
  //TODO Please add an assertion here
await getText(page, ".LoginBox-wrapper .u-textColor--error");
  await validate_error_message_2(errorMsg);
  await Promote_permissive_vendor_release();
  await page.click(".UploadLicenseFile--wrapper > div:nth-of-type(2) .link");
  await waitForText(page, "[class^='u-paddingTop'], [class*=' u-paddingTop']", 'Installing your license');
  await page.click("ul > li:nth-of-type(2) a");
  await waitForText(page, ".Footer-wrapper .u-fontSize--small", `v${testPermissiveTargetKotsVersion} available.`);
  await Promote_restrictive_vendor_release_1();
  await page.click(".WatchDetailPage--wrapper [class^='u-marginRight'], [class*=' u-marginRight'] .replicated-link");
  await waitForText(page, ".pending_download [class^='u-fontSize'], [class*=' u-fontSize']", testVendorRestrictiveReleaseSemver);
  await waitForText(page, "[class^='u-marginBottom'], [class*=' u-marginBottom'] .btn", 'Download');
  await waitForText(page, "[class^='u-marginBottom'], [class*=' u-marginBottom'] .status-tag", 'Pending download');
  //TODO Please add an assertion here
await getText(page, "[class^='u-marginBottom'], [class*=' u-marginBottom'] .u-textColor--error");
  await validate_error_message_3(errorMsg);
  await page.click("[class^='u-marginBottom'], [class*=' u-marginBottom'] .btn");
  await page.waitFor(1000);
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  //TODO Please add an assertion here
await getText(page, "[class^='u-marginBottom'], [class*=' u-marginBottom'] .u-textColor--error");
  await validate_error_message_4(errorMsg);
  await page.click(".u-linkColor");
  // Converting a 'negative-element-validation' step has to be done manually at this time
  // Converting a 'negative-element-validation' step has to be done manually at this time
  // Converting a 'negative-element-validation' step has to be done manually at this time
  await page.click(".justifyContent--spaceBetween .secondary");
  //TODO Please add an assertion here
await isVisible(page, ".pending_download [class^='u-fontSize'], [class*=' u-fontSize']");
  //TODO Please add an assertion here
await isVisible(page, ".pending_download .btn");
  await page.click("ul > li:nth-of-type(5) a");
  await validate_view_files_seqience();
}

