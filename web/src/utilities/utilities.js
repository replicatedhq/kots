// TODO: fix linting issues
/* eslint-disable */
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import timezone from "dayjs/plugin/timezone";
import advanced from "dayjs/plugin/advancedFormat";
import Cookies from "universal-cookie";
import utc from "dayjs/plugin/utc";
import queryString from "query-string";
import sortBy from "lodash/sortBy";
import cronstrue from "cronstrue";
import size from "lodash/size";
import each from "lodash/each";
import find from "lodash/find";
import trim from "lodash/trim";
import * as jsdiff from "diff";
import zlib from "zlib";
import yaml from "js-yaml";
import tar from "tar-stream";
import fileReaderStream from "filereader-stream";
import deploymentStatusIcon from "../assets/deployment-status.svg";

dayjs.extend(timezone);
dayjs.extend(utc);
dayjs.extend(advanced);
dayjs.extend(relativeTime);

/**
 * Takes a system file and returns its content
 *
 * @param {Object} file the system file object
 * @return {String}
 */
export function getFileContent(file) {
  return new Promise((resolve, reject) => {
    let content = "";
    const reader = new FileReader();
    reader.onloadend = function (e) {
      content = e.target.result;
      resolve(content);
    };
    reader.onerror = function (e) {
      reject(e);
    };
    reader.readAsArrayBuffer(file);
  });
}

/**
 * Retrieves the type of application via a watched app's metadata
 *
 * @param {Object} watch The watched application to check
 * @return {String} one of {"replicated.app"|"helm"}
 */
export function getApplicationType(watch) {
  try {
    const { metadata } = watch;
    if (!metadata || metadata === "null") {
      return "";
    }
    const parsedMetadata = JSON.parse(metadata);
    return parsedMetadata.applicationType;
  } catch (error) {
    console.error(error);
    return "Error fetching applicationType";
  }
}

/**
 * @param {String} - Returns the commit SHA of the current build
 */
export function getBuildVersion() {
  return process.env.KOTSADM_BUILD_VERSION;
}

/**
 * @param {Array} - the features flag array
 * @param {String} - name of feature to check
 */
export function isFeatureEnabled(featureArr, featureName) {
  if (!featureArr || featureArr.length === 0) {
    return false;
  }
  return featureArr.includes(featureName);
}

export function parseIconUri(uri) {
  const splitUri = uri.split("?");
  if (splitUri.length < 2) {
    return {
      uri: deploymentStatusIcon,
      dimensions: {
        w: 17,
        h: 17,
      },
    };
  }
  return {
    uri: deploymentStatusIcon,
    dimensions: queryString.parse(splitUri[1]),
  };
}

export function calculateTimeDifference(start, end) {
  const date1 = dayjs(start);
  const date2 = dayjs(end);
  const seconds = date2.diff(date1, "s");
  let formattedDiff;
  if (seconds >= 3600) {
    const hourDiff = date2.diff(date1, "h");
    formattedDiff = `${hourDiff} hour${hourDiff === 1 ? "" : "s"}`;
  } else if (seconds >= 60) {
    const minuteDiff = date2.diff(date1, "m");
    formattedDiff = `${minuteDiff} minute${minuteDiff === 1 ? "" : "s"}`;
  } else {
    formattedDiff = `${seconds} second${seconds === 1 ? "" : "s"}`;
  }

  return formattedDiff;
}

export function secondsAgo(time) {
  const date1 = dayjs(time);
  return dayjs().diff(date1, "s");
}

/**
 * Retrieves the type of application via a watched app's metadata
 *
 * @param {String} text The string you're checking to see if it needs resizing
 * @param {Int}    maxWidth The maximum width of the texts container
 * @param {String} defaultFontSize The default font-size of the string (ex 32px)
 * @param {Int} minFontSize The minimum font-size the string can be (ex 18)
 * @return {String} new font-size for text to fit one line (ex 28px)
 */
export function dynamicallyResizeText(text, maxWidth, defaultFontSize) {
  let size;
  const resizerElm = document.createElement("p");
  resizerElm.textContent = text;
  resizerElm.classList.add("u-fontWeight--bold");
  resizerElm.style.visibility = "hidden";
  resizerElm.style.zIndex = "-1";
  resizerElm.style.position = "absolute";
  resizerElm.style.fontSize = defaultFontSize;
  document.body.appendChild(resizerElm);

  const resizerWidth = () => resizerElm.getBoundingClientRect().width;
  const resizerFontSize = () => parseInt(resizerElm.style.fontSize, 10);
  if (resizerWidth() - maxWidth < 300) {
    resizerElm.remove();
    size = defaultFontSize;
  }

  // if the difference between the resizer width and the max width is greater than 350px, then resize the font
  while (resizerWidth() - maxWidth >= 350) {
    // do not let resizeFontSize go below 24px
    if (resizerFontSize > 24) {
      resizerElm.style.fontSize = `${size - 1}px`;
      size = resizerFontSize;
    } else {
      return `24px`;
    }
  }

  resizerElm.remove();
  if (size < 24) {
    return "24px";
  }
  return size;
}

export function sortAnalyzers(bundleInsight) {
  return sortBy(bundleInsight, (item) => {
    switch (item.severity) {
      case "error":
        return 1;
      case "warn":
        return 2;
      case "info":
        return 3;
      case "debug":
        return 4;
      default:
        return 1;
    }
  });
}

export function getCronInterval(frequency) {
  switch (frequency) {
    case "0 * * * *":
      return "hourly";
    case "0 0 * * *":
      return "daily";
    case "0 0 * * MON":
      return "weekly";
    default:
      return "custom";
  }
}

export function getCronFrequency(schedule) {
  switch (schedule) {
    case "hourly":
      return "0 * * * *";
    case "daily":
      return "0 0 * * *";
    default:
      return "0 0 * * MON";
  }
}

export function getReadableCronDescriptor(expression) {
  if (expression == "@hourly") {
    expression = "0 * * * *";
  } else if (expression == "@daily") {
    expression = "0 0 * * *";
  } else if (expression == "@weekly") {
    expression = "0 0 * * 0";
  }
  return cronstrue.toString(expression);
}

export function getGitOpsUri(
  provider,
  ownerRepo,
  hostname = "",
  httpPort = ""
) {
  const owner = (ownerRepo.split("/").length && ownerRepo.split("/")[0]) || "";
  const repo =
    (ownerRepo.split("/").length > 1 && ownerRepo.split("/")[1]) || "";

  switch (provider) {
    case "github":
      return `https://github.com/${ownerRepo}`;
    case "github_enterprise":
      return `https://${hostname}/${ownerRepo}`;
    case "gitlab":
      return `https://gitlab.com/${ownerRepo}`;
    case "gitlab_enterprise":
      return `https://${hostname}/${ownerRepo}`;
    case "bitbucket":
      return `https://bitbucket.org/${ownerRepo}`;
    case "bitbucket_server":
      return `https://${hostname}:${httpPort}/projects/${owner}/repos/${repo}`;
    default:
      return `https://github.com/${ownerRepo}`;
  }
}

export function getGitOpsServiceSite(provider, hostname = "") {
  switch (provider) {
    case "github":
      return "github.com";
    case "github_enterprise":
      return hostname;
    case "gitlab":
      return "gitlab.com";
    case "gitlab_enterprise":
      return hostname;
    case "bitbucket":
      return "bitbucket.org";
    case "bitbucket_server":
      return hostname;
    default:
      return "github.com";
  }
}

export function getReadableGitOpsProviderName(provider) {
  switch (provider) {
    case "github":
      return "GitHub";
    case "github_enterprise":
      return "GitHub Enterprise";
    case "gitlab":
      return "GitLab";
    case "gitlab_enterprise":
      return "GitLab Enterprise";
    case "bitbucket":
      return "Bitbucket";
    case "bitbucket_server":
      return "Bitbucket Server";
    default:
      return "GitHub";
  }
}

export function getAddKeyUri(gitops, ownerRepo) {
  const gitUri = gitops?.uri;
  const provider = gitops?.provider;
  const hostname = gitops?.hostname;
  const httpPort = gitops?.httpPort;
  const isGitlab = provider === "gitlab" || provider === "gitlab_enterprise";
  const isBitbucket = provider === "bitbucket";
  const isBitbucketServer = provider === "bitbucket_server";

  let addKeyUri = `${gitUri}/settings/keys/new`;
  if (isGitlab) {
    addKeyUri = `${gitUri}/-/settings/repository`;
  } else if (isBitbucket) {
    const owner = ownerRepo.split("/").length && ownerRepo.split("/")[0];
    addKeyUri = `https://bitbucket.org/account/user/${owner}/ssh-keys/`;
  } else if (isBitbucketServer) {
    const project = ownerRepo.split("/").length && ownerRepo.split("/")[0];
    const repo = ownerRepo.split("/").length > 1 && ownerRepo.split("/")[1];
    addKeyUri = `https://${hostname}:${httpPort}/plugins/servlet/ssh/projects/${project}/repos/${repo}/keys`;
  }

  return addKeyUri;
}

export function requiresHostname(provider) {
  return (
    provider === "gitlab_enterprise" ||
    provider === "github_enterprise" ||
    provider === "bitbucket_server" ||
    provider === "other"
  );
}

/**
 * @param {Number} numerator
 * @param {Number} denominator
 * @return {String} danger, warning or check
 */
export function getPercentageStatus(numerator, denominator) {
  if (!numerator || !denominator) {
    return "unknown";
  }
  const percentage = numerator / denominator;
  return percentage < 0.1 ? "danger" : percentage < 0.25 ? "warning" : "check";
}

export function getLicenseExpiryDate(license) {
  if (!license) {
    return "";
  }
  if (
    !license.expiresAt ||
    license.expiresAt === "" ||
    license.expiresAt === "0001-01-01T00:00:00Z"
  ) {
    return "Never";
  }
  return Utilities.dateFormat(license.expiresAt, "MMM D, YYYY", false);
}

export function rootPath(path) {
  if (path[0] !== "/") {
    return (path = `/${path}`);
  }
  return path;
}

export function getFileFormat(selectedFile) {
  if (selectedFile === "") {
    return "text";
  }
  const isYaml =
    selectedFile.includes(".human") ||
    selectedFile.includes(".yaml") ||
    selectedFile.includes(".yml");
  if (selectedFile.includes(".json")) {
    return "json";
  }
  if (isYaml) {
    return "yaml";
  }
  return "text";
}

export function diffContent(oldContent, newContent) {
  let addedLines = 0;
  let removedLines = 0;

  const diffs = jsdiff.diffLines(oldContent, newContent);
  diffs.forEach((part) => {
    if (part.added) {
      addedLines += part.count;
    }
    if (part.removed) {
      removedLines += part.count;
    }
  });

  return {
    addedLines,
    removedLines,
    changes: addedLines + removedLines,
  };
}

/**
 * @param {Watch} watch - watch to determine type
 * @return {Boolean}
 */
export function isHelmChart(watch) {
  return Boolean(watch.helmName);
}

export function parseUpstreamUri(uri) {
  let parsedSlug;
  if (uri.includes("replicated")) {
    const splitUri = uri.split("://");
    parsedSlug = splitUri[1];
  }
  return parsedSlug;
}
/**
 * Returns true if any item in version history is awaiting results
 * from kotsadm operator.
 * @param {Array} versionHistory - Downstream version history for a kots app
 * @return {Boolean}
 */
export function isAwaitingResults(versionHistory) {
  for (const version of versionHistory) {
    switch (version.status) {
      case "pending_preflight":
      case "unknown":
      case "deploying":
        return true;
    }
  }
  return false;
}

export function getPreflightResultState(preflightResults) {
  if (size(preflightResults.errors) > 0) {
    return "fail";
  }
  if (size(preflightResults.results) === 0) {
    return "pass";
  }

  const { results } = preflightResults;
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

export function formatByteSize(bytes) {
  if (bytes < 1024) {
    return `${bytes}b`;
  }
  if (bytes < 1048576) {
    return `${(bytes / 1024).toFixed(2)}kb`;
  }
  if (bytes < 1073741824) {
    return `${(bytes / 1048576).toFixed(2)}mb`;
  }
  return `${(bytes / 1073741824).toFixed(2)}gb`;
}

export function getGitProviderDiffUrl(
  repoUri,
  provider,
  oldCommitHash,
  newCommitHash
) {
  switch (provider) {
    case "github" || "gitlab":
      return `${repoUri}/compare/${oldCommitHash}...${newCommitHash}`;
    case "bitbucket":
      return `${repoUri}/branches/compare/${newCommitHash}..${oldCommitHash}#diff`;
    default:
      return `${repoUri}/compare/${oldCommitHash}...${newCommitHash}`;
  }
}

export function getCommitHashFromUrl(commitUrl) {
  if (!commitUrl) {
    return "";
  }
  const uriParts = commitUrl.split("/");
  if (!uriParts.length) {
    return "";
  }
  return uriParts[uriParts.length - 1];
}

/**
 * Calculate if the version of Velero is compatible with kots
 * @param {SnapshotSettings} snapshotSettings - snapshot configuration object
 * @return {Boolean}
 */
export function isVeleroCorrectVersion(snapshotSettings) {
  if (snapshotSettings?.isVeleroRunning && snapshotSettings?.veleroVersion) {
    const semVer = snapshotSettings.veleroVersion.split(".");

    const majorVer = parseInt(semVer[0].slice(1));
    const minorVer = parseInt(semVer[1]);
    const patchVer = parseInt(semVer[2]);

    if (majorVer !== 1) {
      return false;
    }

    if (minorVer < 5) {
      return false;
    }

    if (minorVer === 5 && patchVer < 1) {
      return false;
    }

    return true;
  }
  return false;
}

/**
 * Get readable label for a snapshot storage destination
 * @param {String} provider - storage provider string
 * @return {String}
 */
export function getSnapshotDestinationLabel(provider) {
  const DESTINATIONS = {
    aws: "Amazon S3",
    azure: "Azure Blob Storage",
    gcp: "Google Cloud Storage",
    other: "Other S3-Compatible Storage",
    internal: "Internal Storage (Default)",
    nfs: "Network File System (NFS)",
    hostpath: "Host Path",
  };
  return DESTINATIONS[provider] || "Unknown storage provider";
}

export const Utilities = {
  getSessionRoles() {
    if (this.localStorageEnabled()) {
      return window.localStorage.getItem("session_roles");
    }
    return null;
  },

  sessionRolesHasOneOf(rolesSet) {
    const sessionRoles = this.getSessionRoles();
    if (!sessionRoles) {
      // rbac is not enabled
      return true;
    }
    for (const r of rolesSet) {
      if (sessionRoles.includes(r)) {
        return true;
      }
    }
    return false;
  },

  localStorageEnabled() {
    const test = "test";
    try {
      localStorage.setItem(test, test);
      localStorage.removeItem(test);
      return true;
    } catch (e) {
      return false;
    }
  },

  getCookie(cname) {
    const cookies = new Cookies();
    return cookies.get(cname);
  },

  removeCookie(cname) {
    const cookies = new Cookies();
    cookies.remove(cname);
  },

  isLoggedIn() {
    if (localStorage.getItem("isLoggedIn") === "true") {
      return true;
    } else {
      return false;
    }
  },

  dateFormat(date, format, localize = true) {
    if (!localize) {
      return dayjs.utc(date).format(format);
    }
    return dayjs.utc(date).local().format(format);
  },

  dateFromNow(date) {
    return dayjs.utc(date).local().fromNow();
  },

  gqlUnauthorized(message) {
    return message === "GraphQL error: Unauthorized";
  },

  getReadableLoginType(type) {
    switch (type) {
      case "gitlab":
        return "GitLab";
      case "bitbucket":
        return "Bitbucket";
      default:
        return "GitHub";
    }
  },

  clusterState(state) {
    switch (state) {
      case "Waiting":
        return "Waiting for a previous upgrade";
      case "Enqueued":
        return "Upgrading";
      case "Installing":
        return "Upgrading";
      case "Installed":
        return "Up to date";
      case "Obsolete":
        return "No active cluster upgrade found";
      case "KubernetesInstalled":
        return "Cluster version upgraded";
      case "AdonsInstalling":
        return "Upgrading addons";
      case "HelmChartUpdateFailure":
        return "Failed to upgrade addons";
      case "Failed":
        return "Failed";
      default:
        return "Unknown";
    }
  },

  isClusterUpgrading(state) {
    const normalizedState = this.clusterState(state);
    return (
      normalizedState === "Upgrading" || normalizedState === "Upgrading addons"
    );
  },

  shouldShowClusterUpgradeModal(apps) {
    if (!apps || apps.length === 0) {
      return false;
    }

    // embedded cluster can only have one app
    const app = apps[0];

    const triedToDeploy =
      app.downstream?.currentVersion?.status === "deploying" ||
      app.downstream?.currentVersion?.status === "deployed" ||
      app.downstream?.currentVersion?.status === "failed";
    if (!triedToDeploy) {
      return false;
    }

    // show the upgrade modal if the user has tried to deploy the current version
    // and the cluster will upgrade or is already upgrading
    return (
      app.downstream?.cluster?.requiresUpgrade ||
      Utilities.isClusterUpgrading(app.downstream?.cluster?.state)
    );
  },

  // Converts string to titlecase i.e. 'hello' -> 'Hello'
  // @returns {String}
  toTitleCase(word) {
    let i;
    let j;
    let str;
    let lowers;
    let uppers;
    const _word = typeof word === "string" ? word : this;
    str = _word.replace(
      /([^\W_]+[^\s-]*) */g,
      (txt) => txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase()
    );

    // Certain minor words should be left lowercase unless
    // they are the first or last words in the string
    lowers = [
      "A",
      "An",
      "The",
      "And",
      "But",
      "Or",
      "For",
      "Nor",
      "As",
      "At",
      "By",
      "For",
      "From",
      "In",
      "Into",
      "Near",
      "Of",
      "On",
      "Onto",
      "To",
      "With",
    ];
    for (i = 0, j = lowers.length; i < j; i++) {
      str = str.replace(new RegExp(`\\s${lowers[i]}\\s`, "g"), (txt) =>
        txt.toLowerCase()
      );
    }

    // Certain words such as initialisms or acronyms should be left uppercase
    uppers = ["Id", "Tv"];
    for (i = 0, j = uppers.length; i < j; i++) {
      str = str.replace(
        new RegExp(`\\b${uppers[i]}\\b`, "g"),
        uppers[i].toUpperCase()
      );
    }

    return str;
  },

  logoutUser(client, options = {}) {
    window.localStorage.removeItem("isLoggedIn");

    const sessionRoles = this.getSessionRoles();
    if (sessionRoles) {
      window.localStorage.removeItem("session_roles");
    }

    const redirectPath = options?.snapshotRestore
      ? "/restore-completed"
      : "/secure-console";
    if (window.location.pathname !== redirectPath) {
      window.location = redirectPath;
    }
  },

  isEmailValid(email) {
    const newEmail = email.trim();
    const exp =
      /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    return exp.test(newEmail);
  },

  snapshotStatusToDisplayName(status) {
    // The front end replacessome status values with user friendly messages
    switch (status) {
      case "PartiallyFailed":
        return "Incomplete (Failed)";
      case "InProgress":
        return "In Progress";
      case "FailedValidation":
        return "Failed Validation";
    }

    return status;
  },

  arrangeIntoTree(paths) {
    const tree = [];
    each(paths, (path) => {
      const pathParts = path.split("/");
      if (pathParts[0] === "") {
        pathParts.shift(); // remove first blank element from the parts array.
      }
      let currentLevel = tree; // initialize currentLevel to root
      let currentPath = "";
      each(pathParts, (part) => {
        currentPath = `${currentPath}/${part}`;
        // check to see if the path already exists.
        const existingPath = find(currentLevel, ["name", part]);
        if (existingPath) {
          // the path to this item was already in the tree, so don't add it again.
          // set the current level to this path's children
          currentLevel = existingPath.children;
        } else {
          const newPart = {
            name: part,
            path: currentPath,
            children: [],
          };
          currentLevel.push(newPart);
          currentLevel = newPart.children;
        }
      });
    });
    return tree;
  },

  arrangeIntoApplicationTree(paths) {
    const tree = this.arrangeIntoTree(paths);
    return sortBy(tree, (file) => {
      switch (file.path) {
        case "/upstream":
          return 1;
        case "/base":
          return 2;
        case "/overlays":
          return 3;
        default:
          return 4;
      }
    });
  },

  bytesToSize(bytes) {
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    if (bytes === 0) {
      return "0 B";
    }
    const i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
    if (i === 0) {
      return `${bytes} ${sizes[i]}`;
    }
    return `${(bytes / 1024 ** i).toFixed(1)} ${sizes[i]}`;
  },

  getDeployErrorTab(tabs) {
    if (trim(tabs.dryrunStderr) !== "") {
      return "dryrunStderr";
    }
    if (trim(tabs.applyStderr) !== "") {
      return "applyStderr";
    }
    return Object.keys(tabs)[0];
  },

  checkIsDateExpired(date) {
    const currentDate = dayjs.utc();
    const expirationDate = dayjs.utc(date);

    return currentDate.isAfter(expirationDate);
  },

  checkIsDeployedConfigLatest(app) {
    const latestSequence = app?.currentSequence;
    const deployedSequence = app?.downstream?.currentVersion?.parentSequence;

    if (deployedSequence === latestSequence) {
      return true;
    }
    return false;
  },

  getFileFromAirgapBundle(bundle, filename) {
    return new Promise((resolve, reject) => {
      try {
        const extract = tar.extract();
        const gzunipStream = zlib.createGunzip();
        fileReaderStream(bundle)
          .pipe(gzunipStream)
          .pipe(extract)
          .on("entry", (header, stream, next) => {
            if (header.name !== filename) {
              stream.on("end", () => {
                next();
              });
              stream.resume();
              return;
            }
            const buffers = [];
            stream.on("data", (buffer) => {
              buffers.push(buffer);
            });
            stream.on("end", async () => {
              resolve(new Blob(buffers));
            });
            stream.resume();
          })
          .on("finish", () => {
            resolve();
          });
      } catch (err) {
        reject(err);
      }
    });
  },

  getAppSpecFromAirgapBundle(bundleArchive) {
    return new Promise((resolve, reject) => {
      try {
        this.getFileFromAirgapBundle(bundleArchive, "app.tar.gz")
          .then((appArchive) => {
            if (!appArchive) {
              resolve();
              return;
            }
            const extract = tar.extract();
            const gzunipStream = zlib.createGunzip();
            fileReaderStream(appArchive)
              .pipe(gzunipStream)
              .pipe(extract)
              .on("entry", (header, stream, next) => {
                if (getFileFormat(header.name) !== "yaml") {
                  stream.on("end", () => {
                    next();
                  });
                  stream.resume();
                  return;
                }
                const buffers = [];
                stream.on("data", (buffer) => {
                  buffers.push(buffer);
                });
                stream.on("end", async () => {
                  try {
                    const content = Buffer.concat(buffers).toString("utf-8");
                    const docs = await yaml.safeLoadAll(content);
                    for (const doc of docs) {
                      if (!doc) {
                        continue;
                      }
                      if (
                        doc?.kind === "Application" &&
                        doc?.apiVersion === "kots.io/v1beta1"
                      ) {
                        resolve(content);
                        return;
                      }
                    }
                  } catch (_) {
                    // invalid yaml file, don't stop
                  }
                  next();
                });
                stream.resume();
              })
              .on("finish", () => {
                resolve();
              });
          })
          .catch((err) => {
            reject(err);
          });
      } catch (err) {
        reject(err);
      }
    });
  },

  getAirgapMetaFromAirgapBundle(bundleArchive) {
    return new Promise((resolve, reject) => {
      try {
        this.getFileFromAirgapBundle(bundleArchive, "airgap.yaml").then(
          (metaFile) => {
            resolve(metaFile.text());
          }
        );
      } catch (err) {
        reject(err);
      }
    });
  },
  licenseTypeTag(licenseType) {
    switch (licenseType) {
      case "prod": {
        return {
          iconName: "dollar-sign",
          iconColor: "success-color",
        };
      }
      case "trial": {
        return {
          iconName: "stopwatch",
          iconColor: "trial-license-icon",
        };
      }
      case "dev": {
        return {
          iconName: "code",
          iconColor: "dev-license-icon",
        };
      }
      case "community": {
        return {
          iconName: "user-outline",
          iconColor: "warning-color",
        };
      }
      default: {
        return {
          iconName: "",
          iconColor: "",
        };
      }
    }
  },
};
