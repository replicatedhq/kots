"use strict";

const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({headless: false});
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto("http://localhost:8080");
  await Install__configure__deploy();
  await upload__configure__deploy();
  await validate_airgap_smoke_test_files();
  await Validate_Airgap_Support_Bundle();
  await Logout();
  await browser.close();
})();

// move to utils.js

async function waitForText(page, selector, expectedText) {
  await page.waitForFunction(([selector, expectedText]) => {
    const element = document.querySelector(selector);
    return element && element.textContent.replace(/[\r\n]+/g, "").trim() === expectedText.trim();
  }, [selector, expectedText]);
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
async function scrollToElement(page, selector) {
  await page.evaluate((selector) => {
    const element = document.querySelector(selector);
    element.scrollIntoView({ block: "center", inline: "nearest", behavior: "instant" });
  }, selector);
}


// shared steps \\
async function Install__configure__deploy() {
  await page.click("[type='password']");
  await page.type("[type='password']", 'password');
  await page.click("[type='submit']");
  await waitForText(page, ".u-fontSize--header", 'Upload your license file');
  await page.waitForSelector("[class^='u-marginTop'], [class*=' u-marginTop'] .u-textAlign--center", {state: "visible"});
  // Converting a 'drop-file' step has to be done manually at this time
  await page.click("[type='button']");
  await page.waitForSelector(".login-form-wrapper", {state: "visible"});
  await page.waitForSelector(".login-form-wrapper", {state: "visible"});
  await waitForText(page, "form > :nth-child(1) .u-fontSize--normal", 'Hostname');
  await scrollToElement(page, ".u-textDecoration--underlineOnHover");
  await page.click(".u-textDecoration--underlineOnHover");
  await page.waitFor(1000);
  // Converting a 'drop-file' step has to be done manually at this time
  await page.click("[type='button']");
  //TODO Please add an assertion here
await isVisible(page, ".u-fontSize--larger");
  //TODO Please add an assertion here
await isVisible(page, ".card-title");
  //TODO Please add an assertion here
await isVisible(page, "[id='example_config_default-group'] p");
  //TODO Please add an assertion here
await isVisible(page, ".default-value-section");
  //TODO Please add an assertion here
await isVisible(page, "[id='example_config_required-group'] p");
  //TODO Please add an assertion here
await isVisible(page, ".field-label");
  await page.click("[id='example_config_required-group'] [type='text']");
  await page.type("[id='example_config_required-group'] [type='text']", 'some value');
  await page.waitFor(3000);
  await page.click(".btn");
  await waitForText(page, ".justifyContent--space-between .u-textColor--primary", 'Required Kubernetes Version');
  await page.click("[data-tip-disable='true']");
  //TODO Please add an assertion here
await isVisible(page, "[class^='u-paddingRight'], [class*=' u-paddingRight'] .dashboard-card");
  await waitForText(page, "[class^='u-marginLeft'], [class*=' u-marginLeft'] .u-fontSize--normal", 'Ready');
}

async function upload__configure__deploy() {
  await page.click(".WatchDetailPage--wrapper .is-active a");
  await page.waitForSelector("[id='mount-aware-wrapper'] .replicated-link", {state: "visible"});
  await page.click("[id='mount-aware-wrapper'] .replicated-link");
  await page.waitFor(3000);
  // Converting a 'input-file' step has to be done manually at this time
  await page.waitForSelector("[class^='u-paddingRight'], [class*=' u-paddingRight'] .u-fontSize--normal", {state: "visible"});
  //TODO Please add an assertion here
await isVisible(page, "[class^='u-marginTop'], [class*=' u-marginTop'] .btn");
  await page.click("[data-tip-disable='true']");
  //TODO Please add an assertion here
await isVisible(page, "[id='example_config_required_2-group'] p");
  //TODO Please add an assertion here
await isVisible(page, "[id='example_config_required_2-group'] .field-label");
  await page.click("[id='example_config_required_2-group'] [type='text']");
  await page.type("[id='example_config_required_2-group'] [type='text']", 'some other value');
  //TODO Please add an assertion here
await isVisible(page, "#ports > h3");
  //TODO Please add an assertion here
await isVisible(page, "[id='serviceport-group'] .card-item-title p");
  //TODO Please add an assertion here
await isVisible(page, ".add-btn");
  await page.click(".add-btn");
  await page.waitFor(3000);
  //TODO Please add an assertion here
await isVisible(page, "[id='ports'] .config-items > :nth-child(2) .card-item-title p");
  //TODO Please add an assertion here
await isVisible(page, "[id='ports'] .config-items > :nth-child(2) .Input");
  await page.click("[id='ports'] .config-items > :nth-child(2) .Input");
  await page.type("[id='ports'] .config-items > :nth-child(2) .Input", '443');
  await page.waitFor(3000);
  await page.click(".btn");
  await page.click("body > :nth-child(10) .primary");
  //TODO Please add an assertion here
await isVisible(page, ".version > :nth-child(2) .pending .u-textColor--lightAccent");
  await waitForText(page, ".version > :nth-child(2) .checks-running-text", 'Checks passed');
  //TODO Please add an assertion here
await isVisible(page, ".version > :nth-child(2) .primary");
  await page.click(".version > :nth-child(2) .pending [data-tip-disable='true']");
  await page.waitFor(1000);
  await page.click("body > :nth-child(13) [class^='u-marginLeft'], [class*=' u-marginLeft']");
  //TODO Please add an assertion here
await isVisible(page, ".success");
  await page.click(".WatchDetailPage--wrapper ul > li:nth-of-type(1) a");
  //TODO Please add an assertion here
await isVisible(page, ".u-textColor--lightAccent");
  //TODO Please add an assertion here
await isVisible(page, ".status-tag");
  await waitForText(page, ".status-tag", 'Currently deployed version');
  await page.waitForSelector("[class^='u-marginLeft'], [class*=' u-marginLeft'] .u-fontSize--normal", {state: "visible"});
  await waitForText(page, "[class^='u-marginLeft'], [class*=' u-marginLeft'] .u-fontSize--normal", 'Ready');
}

async function validate_airgap_smoke_test_files() {
  await page.click(".tw-relative ul > li:nth-of-type(6) a");
  await waitForText(page, "[for='sub-dir-upstream-6-/upstream-0-0']", 'upstream');
  await waitForText(page, "[for='sub-dir-base-3-/base-1-0']", 'base');
  await waitForText(page, "[for='sub-dir-overlays-2-/overlays-2-0']", 'overlays');
  await waitForText(page, "[for='sub-dir-kotsKinds-5-/kotsKinds-3-0']", 'kotsKinds');
  await waitForText(page, "[for='sub-dir-rendered-1-/rendered-4-0']", 'rendered');
  await page.click(".details-subnav ul > li:nth-of-type(1) a");
}

async function Validate_Airgap_Support_Bundle() {
  await page.click(".tw-relative ul > li:nth-of-type(4) a");
  await page.click(".btn");
  await waitForText(page, ".tab-items > :nth-child(2)", 'File inspector');
  await page.click(".tab-items > :nth-child(2)");
  //TODO Please add an assertion here
await isVisible(page, "[for='sub-dir-secrets-1-secrets-4-0']");
  await waitForText(page, "[for='sub-dir-secrets-1-secrets-4-0']", 'secrets');
  await page.click("[for='sub-dir-secrets-1-secrets-4-0']");
  //TODO Please add an assertion here
await isVisible(page, "[for='sub-dir-airgap-smoke-test-4-secrets/airgap-smoke-test-0-1']");
  await waitForText(page, "[for='sub-dir-airgap-smoke-test-4-secrets/airgap-smoke-test-0-1']", 'airgap-smoke-test');
  await page.click("[for='sub-dir-airgap-smoke-test-4-secrets/airgap-smoke-test-0-1']");
  //TODO Please add an assertion here
await isVisible(page, "[for='sub-dir-kotsadm-airgap-smoke-test-instance-report-1-secrets/airgap-smoke-test/kotsadm-airgap-smoke-test-instance-report-1-2']");
  await waitForText(page, "[for='sub-dir-kotsadm-airgap-smoke-test-instance-report-1-secrets/airgap-smoke-test/kotsadm-airgap-smoke-test-instance-report-1-2']", 'kotsadm-airgap-smoke-test-instance-report');
  await page.click("[for='sub-dir-kotsadm-airgap-smoke-test-instance-report-1-secrets/airgap-smoke-test/kotsadm-airgap-smoke-test-instance-report-1-2']");
  //TODO Please add an assertion here
await isVisible(page, ".FileTree-wrapper > :nth-child(5) > :nth-child(3) > :nth-child(1) > :nth-child(3) > :nth-child(2) div");
  await waitForText(page, ".FileTree-wrapper > :nth-child(5) > :nth-child(3) > :nth-child(1) > :nth-child(3) > :nth-child(2) div", 'report.json');
  //TODO Please add an assertion here
await isVisible(page, "[for='sub-dir-kotsadm-airgap-smoke-test-preflight-report-1-secrets/airgap-smoke-test/kotsadm-airgap-smoke-test-preflight-report-2-2']");
  await waitForText(page, "[for='sub-dir-kotsadm-airgap-smoke-test-preflight-report-1-secrets/airgap-smoke-test/kotsadm-airgap-smoke-test-preflight-report-2-2']", 'kotsadm-airgap-smoke-test-preflight-report');
  await page.click("[for='sub-dir-kotsadm-airgap-smoke-test-preflight-report-1-secrets/airgap-smoke-test/kotsadm-airgap-smoke-test-preflight-report-2-2']");
  //TODO Please add an assertion here
await isVisible(page, ".FileTree-wrapper > :nth-child(5) > :nth-child(3) > :nth-child(1) > :nth-child(3) > :nth-child(3) div");
  await waitForText(page, ".FileTree-wrapper > :nth-child(5) > :nth-child(3) > :nth-child(1) > :nth-child(3) > :nth-child(3) div", 'report.json');
  //TODO Please add an assertion here
await isVisible(page, "[for='sub-dir-replicated-instance-report-1-secrets/airgap-smoke-test/replicated-instance-report-4-2']");
  await waitForText(page, "[for='sub-dir-replicated-instance-report-1-secrets/airgap-smoke-test/replicated-instance-report-4-2']", 'replicated-instance-report');
  await page.click("[for='sub-dir-replicated-instance-report-1-secrets/airgap-smoke-test/replicated-instance-report-4-2']");
  //TODO Please add an assertion here
await isVisible(page, ".FileTree-wrapper > :nth-child(8) > :nth-child(3) > :nth-child(1) > :nth-child(3) > :nth-child(5) div");
  await waitForText(page, ".FileTree-wrapper > :nth-child(8) > :nth-child(3) > :nth-child(1) > :nth-child(3) > :nth-child(5) div", 'report.json');
  await page.click(".details-subnav ul > li:nth-of-type(1) a");
}

async function Logout() {
  await page.click(".navbar-dropdown-container span");
  await page.click("[data-qa='Navbar--logOutButton']");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  //TODO Please add an assertion here
await isVisible(page, "[type='password']");
  await waitForText(page, "[type='submit']", 'Log in');
}

