import yaml from "js-yaml";
import * as _ from "lodash";
import { KLicense, KEntitlement } from "../klicenses";
import { KotsApp } from "../kots_app";
import { ReplicatedError } from "../server/errors";

const jsdiff = require('diff');

export function base64Decode(data: string): string {
  if (!data) {
    return "";
  }
  const buffer = new Buffer(data, 'base64');
  return buffer.toString("ascii");
}

export function base64Encode(value: string): string {
  return Buffer.from(value).toString("base64");
};

export function getPreflightResultState(preflightResults): string {
  if (_.size(preflightResults.errors) > 0) {
    return "fail";
  }
  if (_.size(preflightResults.results) === 0) {
    return "pass";
  }

  const results = preflightResults.results;
  let resultState = "pass";
  for (const check of results) {
    if (check.isWarn) {
      resultState = "warn";
    } else if (check.isFail) {
      return "fail";
    }
  }
  return resultState;
}

export function getLicenseInfoFromYaml(licenseData): KLicense {
  try {
    const licenseJson = yaml.safeLoad(licenseData);
    const spec = licenseJson.spec;
  
    const license = new KLicense();
    license.id = spec.licenseID;
    license.channelName = spec.channelName || "";

    license.expiresAt = "";
    if (spec.entitlements && spec.entitlements.expires_at) {
      license.expiresAt = spec.entitlements.expires_at.value;
    }

    if (spec.licenseSequence) {
      license.licenseSequence = spec.licenseSequence;
    }

    if (spec.licenseType) {
      license.licenseType = spec.licenseType;
    }

    const entitlements: KEntitlement[] = [];
    if (spec.entitlements) {
      const keys = Object.keys(spec.entitlements);
      for (let k = 0; k < keys.length; k++) {
        const key = keys[k];
        const entitlement = spec.entitlements[key];
        if (!entitlement.isHidden && key !== "expires_at") {
          entitlements.push({
            title: entitlement.title,
            value: entitlement.value,
            label: key,
          });
        }
      }
    }
    license.entitlements = entitlements;
  
    return license;
  } catch(err) {
    throw new ReplicatedError(`Error getting license info from yaml file ${err}`);
  }
}

export async function getDiffSummary(app: KotsApp): Promise<string> {
  if (!app || app.currentSequence == undefined) {
    return "";
  }
  const oldSequence = `${app.currentSequence}`;
  const newSequence = `${app.currentSequence + 1}`;

  const oldPaths = await app.getFilesPaths(oldSequence);
  const oldFiles = await app.getFiles(oldSequence, oldPaths);

  const newPaths = await app.getFilesPaths(newSequence);
  const newFiles = await app.getFiles(newSequence, newPaths);

  let filesChanged = 0, linesAdded = 0, linesRemoved = 0;
  for (const path in oldFiles.files) {
    if (!(path in newFiles.files)) {
      filesChanged++;
      linesRemoved += oldFiles.files[path].toString().split("\n").length;
      continue;
    }
    const oldContent: Buffer = oldFiles.files[path];
    const newContent: Buffer = newFiles.files[path];
    const diffs = jsdiff.diffLines(oldContent.toString(), newContent.toString());

    let fileHasChanged = false;
    diffs.forEach(part => {
      if (part.added) {
        fileHasChanged = true;
        linesAdded += part.count;
      }
      if (part.removed) {
        fileHasChanged = true;
        linesRemoved += part.count;
      }
    });

    if (fileHasChanged) {
      filesChanged++;
    }
  }

  for (const path in newFiles.files) {
    if (!(path in oldFiles.files)) {
      filesChanged++;
      linesAdded += newFiles.files[path].toString().split("\n").length;
    }
  }

  return JSON.stringify({
    filesChanged,
    linesAdded,
    linesRemoved
  });
}

export function getLicenseType(license: string): string {
  if (license) {
    const doc = yaml.safeLoad(license.toString());
    if (doc.spec && doc.spec.licenseType) {
      return doc.spec.licenseType;
    } else {
      return "";
    }
  } else {
    return "";
  }
}

export function getGitProviderCommitUrl(repoUri: string, commitHash: string, provider: string): string {
  switch (provider) {
    case "github" || "github_enterprise" || "gitlab":
      return `${repoUri}/commit/${commitHash}`;
    case "bitbucket":
      return `${repoUri}/commits/${commitHash}`;
    default:
      return `${repoUri}/commit/${commitHash}`;
  }
}

export async function sleep(seconds): Promise<void> {
  await new Promise(resolve => setTimeout(resolve, seconds * 1000));
}
