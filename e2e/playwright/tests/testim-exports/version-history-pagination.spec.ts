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
  // Converting a 'input-file' step has to be done manually at this time
  await page.click("[type='button']");
  await waitForText(page, ".u-fontSize--larger", 'Configure Version History Pagination');
  await page.click(".primary");
  await waitForText(page, "[class^='u-marginLeft'], [class*=' u-marginLeft'] .u-fontSize--normal", 'Ready');
  // Converting a 'cli-action-code-step' step has to be done manually at this time
  // Converting a 'cli-action-code-step' step has to be done manually at this time
  await page.click("ul > li:nth-of-type(2) a");
  await waitForText(page, ".u-marginTop--30 .u-fontSize--normal", 'All versions');
  await waitForText(page, ".version > :nth-child(1) > :nth-child(1) [class^='u-paddingRight'], [class*=' u-paddingRight'] [class^='u-marginLeft'], [class*=' u-marginLeft']", `Sequence ${testLatestSequence}`);
  await waitForText(page, "[class^='u-marginBottom'], [class*=' u-marginBottom'] .VersionHistoryRow .u-textColor--bodyCopy", `Sequence ${testLatestSequence}`);
  await scrollOnElement(page, "[class^='u-padding'], [class*=' u-padding']", 0, 5000);
  //TODO Please add an assertion here
await getText(page, ".u-textAlign--center");
  await validate_pager_text();
  await page.click(".u-display--inlineBlock");
  await waitForText(page, ".version > :nth-child(1) > :nth-child(2) > div:nth-of-type(21) [class^='u-paddingRight'], [class*=' u-paddingRight'] [class^='u-marginLeft'], [class*=' u-marginLeft']", `Sequence ${testNumOfVersions - testDefaultPageSize * 2}`);
  //TODO Please add an assertion here
await getText(page, ".u-textAlign--center");
  await validate_pager_text_1();
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

function getText(page, selector) {
  return page.evaluate((selector) => {
    const element = document.querySelector(selector);
    return element.textContent;
  }, selector);
}


// shared steps \\
async function export_test_vars() {
  await page.evaluate(() => {
    exportsTest.testAppSlug = "version-history-pagination";
    exportsTest.testNamespace = "version-history-pagination";
    exportsTest.testNumOfVersions = 251;
    exportsTest.testLatestSequence = 250;
    exportsTest.testDefaultPageSize = 20;
    exportsTest.testSecondPageFirstSequence = 230;
    
  });
}

async function validate_pager_text() {
  await page.evaluate(() => {
    return pagerText == `Showing releases 1 - ${testDefaultPageSize} of ${testNumOfVersions}`;
    
  });
}

async function validate_pager_text_1() {
  await page.evaluate(() => {
    return pagerText == `Showing releases ${testDefaultPageSize + 1} - ${testDefaultPageSize * 2} of ${testNumOfVersions}`;
    
  });
}

