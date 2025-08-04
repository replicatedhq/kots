import * as core from '@actions/core';

const dateObj = new Date();
const month = dateObj.getUTCMonth() + 1; // months are 0-based
const day = dateObj.getUTCDate();
const year = dateObj.getUTCFullYear();
const sha = process.env.GITHUB_SHA.substring(0, 6);

core.setOutput("GIT_TAG", `v${year}.${month}.${day}-${sha}-nightly`);
