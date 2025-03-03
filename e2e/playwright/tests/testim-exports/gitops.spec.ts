"use strict";

const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({headless: false});
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto("http://localhost:8800");
  await Login_and_Upload_License();
  await Create_a_new_version_with_Config_change();
  await Configure_GitOps();
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
function getText(page, selector) {
  return page.evaluate((selector) => {
    const element = document.querySelector(selector);
    return element.textContent;
  }, selector);
}


// shared steps \\
async function Login_and_Upload_License() {
  await page.click("[type='password']");
  await page.type("[type='password']", 'password');
  await page.click("[type='submit']");
  await waitForText(page, ".u-fontSize--header", 'Upload your license file');
  await page.waitForSelector("[class^='u-marginTop'], [class*=' u-marginTop'] .u-textAlign--center", {state: "visible"});
  // Converting a 'drop-file' step has to be done manually at this time
  await page.click("[type='button']");
  await waitForText(page, "h3", 'Application Configuration');
  await page.waitForSelector(".btn", {state: "visible"});
  await page.click(".btn");
  //TODO Please add an assertion here
await isVisible(page, "[class^='u-paddingRight'], [class*=' u-paddingRight'] .dashboard-card");
  await waitForText(page, "[class^='u-marginLeft'], [class*=' u-marginLeft'] .u-fontSize--normal", 'Ready');
}

async function Create_a_new_version_with_Config_change() {
  await page.click(".tw-relative ul > li:nth-of-type(3) a");
  await waitForText(page, "h3", 'Application Configuration');
  await page.click("#trivial_config");
  await page.waitFor(3000);
  await page.click(".btn");
  await page.waitForSelector(".Modal-body", {state: "visible"});
  await page.click("body > :nth-child(10) .primary");
  await waitForText(page, "[class^='u-marginBottom'], [class*=' u-marginBottom'] .u-textColor--lightAccent", 'Config Change');
  await waitForText(page, "[class^='u-marginBottom'], [class*=' u-marginBottom'] [data-tip-disable='true']", 'Deploy');
}

async function export_gitops_variables() {
  await page.evaluate(() => {
    exports.gitopsOwner = 'replicated-testim-kotsadm-gitops';
    exports.gitopsRepo = 'qakots-kotsadm-gitops';
    exports.githubToken = 'TODO_ADD_GITHUB_TOKEN_FROM_SECRET';
    exports.testAppSlug = 'gitops-bobcat';
    
  });
}

async function add_ssh_key_to_github(sshKeyName, sshKeyValue, githubToken) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function update_config() {
  await page.click(".tw-relative ul > li:nth-of-type(3) a");
  await waitForText(page, "h3", 'Application Configuration');
  await page.click("#trivial_config");
  await page.waitFor(3000);
  await page.click(".btn");
  await page.waitForSelector(".Modal-body", {state: "visible"});
  await page.click("[type='button']");
  await page.waitFor(3000);
}

async function validate_commits_in_github(gitopsOwnerRepo, gitopsBranchName, githubToken) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function validate_content_in_github(gitopsOwnerRepo, gitopsBranchName, gitopsPath, githubToken) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function delete_github_branch(gitopsOwnerRepo, gitopsBranchName, githubToken) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function remove_ssh_key_from_github(sshKeyID, githubToken) {
  // Converting a 'api-action' step has to be done manually at this time
}

async function Validate_Version_History() {
  await page.waitForSelector(".secondary", {state: "visible"});
  await page.click("ul > li:nth-of-type(2) a");
  await waitForText(page, ".version > :nth-child(2) > :nth-child(2) .u-textColor--lightAccent", 'Config Change');
  await waitForText(page, ".version > :nth-child(2) > :nth-child(2) [data-tip-disable='true']", 'Deploy');
  await waitForText(page, ".version > :nth-child(2) > :nth-child(3) .u-textColor--lightAccent", 'Config Change');
  await waitForText(page, ".version > :nth-child(2) > :nth-child(3) [data-tip-disable='true']", 'Deploy');
  await waitForText(page, ".deployed .u-textColor--lightAccent", 'Online Install');
  await waitForText(page, ".deployed [data-tip-disable='true']", 'Redeploy');
}

async function Disable_GitOps() {
  await delete_github_branch(gitopsOwner + "/" + gitopsRepo, gitopsBranchName, githubToken);
  await remove_ssh_key_from_github(sshKeyID, githubToken);
  await page.click(".left-items > div:nth-of-type(2) .flex");
  await page.click(".ClusterDashboard--wrapper a");
  //TODO Please add an assertion here
await isVisible(page, ".u-fontSize--largest");
  await page.click("body > :nth-child(10) .primary");
  // Converting a 'negative-element-validation' step has to be done manually at this time
  await page.click(".left-items > div:nth-of-type(1) .flex");
  await Validate_Version_History();
}

async function Test_Failed_GitOps_Connection() {
  await page.click(".left-items > div:nth-of-type(2) .text span");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  //TODO Please add an assertion here
await isVisible(page, "[class^='u-marginBottom'], [class*=' u-marginBottom']");
  //TODO Please add an assertion here
await isVisible(page, "[class^='css'], [class*=' css'] div > :nth-child(2)");
  await page.click("[placeholder='owner']");
  await page.type("[placeholder='owner']", gitopsOwner);
  await page.click("[placeholder='Repository']");
  await page.type("[placeholder='Repository']", gitopsRepo);
  // Converting a 'random-value-generator' step has to be done manually at this time
  await page.click("[placeholder='main']");
  await page.type("[placeholder='main']", gitopsBranchName);
  // Converting a 'random-value-generator' step has to be done manually at this time
  await page.click("[placeholder='/path/to-deployment']");
  await page.type("[placeholder='/path/to-deployment']", gitopsPath);
  await page.click(".btn");
  await page.waitForSelector(".react-prism", {state: "visible"});
  await page.click(".primary");
  await waitForText(page, ".u-fontSize--largest", 'Connection to repository failed');
  await page.click("[class^='u-marginRight'], [class*=' u-marginRight']");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  await page.click(".secondary");
  await waitForText(page, "[placeholder='owner']", gitopsOwner);
  await waitForText(page, "[placeholder='Repository']", gitopsRepo);
  await page.click(".btn");
  await page.waitForSelector(".react-prism", {state: "visible"});
}

async function Configure_GitOps() {
  await export_gitops_variables();
  await page.click(".left-items > div:nth-of-type(2) .text span");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  //TODO Please add an assertion here
await isVisible(page, "[class^='u-marginBottom'], [class*=' u-marginBottom']");
  //TODO Please add an assertion here
await isVisible(page, "[class^='css'], [class*=' css'] div > :nth-child(2)");
  await page.click("[placeholder='owner']");
  await page.type("[placeholder='owner']", gitopsOwner);
  await page.click("[placeholder='Repository']");
  await page.type("[placeholder='Repository']", gitopsRepo);
  // Converting a 'random-value-generator' step has to be done manually at this time
  await page.click("[placeholder='main']");
  await page.type("[placeholder='main']", gitopsBranchName);
  // Converting a 'random-value-generator' step has to be done manually at this time
  await page.click("[placeholder='/path/to-deployment']");
  await page.type("[placeholder='/path/to-deployment']", gitopsPath);
  await page.click(".btn");
  await page.waitForSelector(".react-prism", {state: "visible"});
  //TODO Please add an assertion here
await isVisible(page, ".CodeSnippet-copy");
  //TODO Please add an assertion here
await getText(page, ".react-prism");
  // Converting a 'random-value-generator' step has to be done manually at this time
  await add_ssh_key_to_github(sshKeyName, sshKeyValue, githubToken);
  await page.click(".primary");
  // Converting a 'wait-for-negative-element-validation' step has to be done manually at this time
  await waitForText(page, ".u-fontSize--largest", 'GitOps is enabled');
  await page.click("body > :nth-child(11) .primary");
  await page.click("ul > li:nth-of-type(2) a");
  // Converting a 'negative-element-validation' step has to be done manually at this time
  // Converting a 'negative-element-validation' step has to be done manually at this time
  await page.waitFor(10000);
  await update_config();
  await validate_commits_in_github(gitopsOwner + "/" + gitopsRepo, gitopsBranchName, githubToken);
  await validate_content_in_github(gitopsOwner + "/" + gitopsRepo, gitopsBranchName, gitopsPath, githubToken);
  await Disable_GitOps();
  await Test_Failed_GitOps_Connection();
}

