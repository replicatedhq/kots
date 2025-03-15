import { Page } from '@playwright/test';

import {
  runCommand,
  runCommandWithOutput,
  RegistryInfo
} from './cli';

export const validateImageGarbageCollection = async (page: Page, registryInfo: RegistryInfo, namespace: string) => {
  await checkImagesBeforeGarbageCollection(registryInfo);
  await runImageGarbageCollection(namespace);
  await page.waitForTimeout(10000);
  await checkImagesAfterGarbageCollection(registryInfo);
};

const checkImagesBeforeGarbageCollection = async (registryInfo: RegistryInfo) => {
  const output = runCommandWithOutput(`curl -k https://${registryInfo.username}:${registryInfo.password}@${registryInfo.ip}/v2/qakotsregression/nginx/tags/list`);
  console.log("found tags before gc", output);

  const parsed = JSON.parse(output);
  // GC will run on app upgrade, so there may be one or two images at this point
  return parsed.tags.length == 2 || parsed.tags.length == 1;
};

const runImageGarbageCollection = async (namespace: string) => {
  runCommand(`kubectl kots admin-console garbage-collect-images -n ${namespace} --ignore-rollback`);
};

const checkImagesAfterGarbageCollection = async (registryInfo: RegistryInfo) => {
  const output = runCommandWithOutput(`curl -k https://${registryInfo.username}:${registryInfo.password}@${registryInfo.ip}/v2/qakotsregression/nginx/tags/list`)
  console.log("found tags after gc", output);

  const parsed = JSON.parse(output);
  return parsed.tags.length == 1;
}
