import * as _ from "lodash";
import { KotsApp } from "../kots_app";

var jsdiff = require('diff');

export function decodeBase64(data: string): string {
  if (!data) {
    return "";
  }
  const buffer = new Buffer(data, 'base64');
  return buffer.toString("ascii");
}

export function getPreflightResultState(preflightResults): string {
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
      linesRemoved += oldFiles.files[path].split("\n").length;
      continue;
    }
    const oldContent = oldFiles.files[path];
    const newContent = newFiles.files[path];
    const diffs = jsdiff.diffLines(oldContent, newContent);

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
      linesAdded += newFiles.files[path].split("\n").length;
    }
  }

  return JSON.stringify({
    filesChanged,
    linesAdded,
    linesRemoved
  });
}
