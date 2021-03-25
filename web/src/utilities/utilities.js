import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import timezone from "dayjs/plugin/timezone";
import advanced from "dayjs/plugin/advancedFormat";
import Cookies from 'universal-cookie';
import utc from "dayjs/plugin/utc";
import queryString from "query-string";
import sortBy from "lodash/sortBy";
import cronstrue from "cronstrue";
import size from "lodash/size";
import each from "lodash/each";
import find from "lodash/find";
import trim from "lodash/trim";
import * as jsdiff from "diff";

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
    if (!metadata || metadata === "null") return "";
    const parsedMetadata = JSON.parse(metadata);
    return parsedMetadata.applicationType;

  } catch (error) {
    console.error(error);
    return "Error fetching applicationType";
  }
}

export function getReadableCollectorName(name) {
  const namesObj = {
    "cluster-info": "Gathering basic information about the cluster",
    "cluster-resources": "Gathering available resources in cluster",
    "mysql": "Gathering information about MySQL",
    "postgres": "Gathering information about PostgreSQL",
    "redis": "Gathering information about Redis"
  }
  const statusToReturn = namesObj[name];
  if (statusToReturn) {
    return statusToReturn
  } else {
    return "Gathering details about the cluster";
  }
}

/**
 * @param {String} - Returns the commit SHA of the current build
 */
export function getBuildVersion() {
  return window.env.KOTSADM_BUILD_VERSION;
}

export function parseIconUri(uri) {
  const splitUri = uri.split("?");
  if (splitUri.length < 2) {
    return {
      uri: "https://troubleshoot.sh/images/analyzer-icons/gray-checkmark.svg",
      dimensions: {
        w: 17,
        h: 17
      }
    };
  }
  return {
    uri: splitUri[0],
    dimensions: queryString.parse(splitUri[1])
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
export function dynamicallyResizeText(text, maxWidth, defaultFontSize, minFontSize) {
  let size;
  let resizerElm = document.createElement("p");
  resizerElm.textContent = text;
  resizerElm.setAttribute("class", "u-fontWeight--bold");
  resizerElm.setAttribute("style", `visibility: hidden; z-index: -1; position: absolute; font-size: ${defaultFontSize}`);
  document.body.appendChild(resizerElm);

  if (resizerElm.getBoundingClientRect().width < maxWidth) {
    resizerElm.remove();
    return defaultFontSize;
  }

  while(resizerElm.getBoundingClientRect().width > maxWidth) {
    size = parseInt(resizerElm.style.fontSize, 10);
    resizerElm.style.fontSize = `${size - 1}px`;
  }

  resizerElm.remove();
  if (minFontSize && size < minFontSize) {
    return `${minFontSize}px`;
  } else {
    // Font size needs to be 1px smaller than the last calculated size to fully fit in the container
    return `${size - 1}px`;
  }
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
  })
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
      return "0 * * * *"
    case "daily":
      return "0 0 * * *";
    default:
      return "0 0 * * MON";
  }
}

export function getReadableCronDescriptor(expression) {
  if (expression == "@hourly") {
    expression = "0 * * * *"
  } else if (expression == "@daily") {
    expression = "0 0 * * *"
  } else if (expression == "@weekly") {
    expression = "0 0 * * 0"
  }
  return cronstrue.toString(expression);
}

export function getServiceSite(provider, hostname = "") {
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
    default:
      return "github.com";
  }
}

export function getAddKeyUri(gitUri, provider, ownerRepo) {
  const isGitlab = provider === "gitlab" || provider === "gitlab_enterprise";
  const isBitbucket = provider === "bitbucket" || provider === "bitbucket_server";

  let addKeyUri = `${gitUri}/settings/keys/new`;
  if (isGitlab) {
    addKeyUri = `${gitUri}/-/settings/repository`;
  } else if (isBitbucket) {
    const owner = ownerRepo.split("/").length && ownerRepo.split("/")[0];
    addKeyUri = `https://bitbucket.org/account/user/${owner}/ssh-keys/`;
  }
  return addKeyUri;
}

export function requiresHostname(provider) {
  return provider === "gitlab_enterprise" || provider === "github_enterprise" || provider === "bitbucket_server" || provider === "other";
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
  if (!license.expiresAt || license.expiresAt === "" || license.expiresAt === "0001-01-01T00:00:00Z") {
    return "Never";
  }
  return Utilities.dateFormat(license.expiresAt, "MMM D, YYYY", false);
}

export function rootPath(path) {
  if (path[0] !== "/") {
    return path = "/" + path;
  } else {
    return path;
  }
}

export function getFileFormat(selectedFile) {
  if (selectedFile === "") {
    return "text";
  }
  const isYaml = selectedFile.includes(".human") || selectedFile.includes(".yaml") || selectedFile.includes(".yml");
  if (selectedFile.includes(".json")) {
    return "json";
  } else if (isYaml) {
    return "yaml";
  }
  return "text";
}

export function diffContent(oldContent, newContent) {
  let addedLines = 0, removedLines = 0;

  const diffs = jsdiff.diffLines(oldContent, newContent);
  diffs.forEach(part => {
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
    changes: addedLines + removedLines
  }
}

/**
 * @param {Watch} watch - watch to determine type
 * @return {Boolean}
 */
export function isHelmChart(watch) {
  return Boolean(watch.helmName);
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
        return true
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

export function formatByteSize(bytes) {
  if (bytes < 1024) {
    return bytes + "b";
  } else if (bytes < 1048576) {
    return (bytes / 1024).toFixed(2) + "kb";
  } else if (bytes < 1073741824) {
    return (bytes / 1048576).toFixed(2) + "mb";
  } else {
    return (bytes / 1073741824).toFixed(2) + "gb";
  }
}

export function getGitProviderDiffUrl(repoUri, provider, oldCommitHash, newCommitHash) {
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

export const Utilities = {
  getToken() {
    if (this.localStorageEnabled()) {
      return window.localStorage.getItem("token");
    } else {
      return "";
    }
  },

  getSessionRoles() {
    if (this.localStorageEnabled()) {
      return window.localStorage.getItem("session_roles");
    } else {
      return null;
    }
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
    var test = "test";
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
    cookies.remove(cname)
  },

  isLoggedIn() {
    const hasToken = this.getToken();
    return !!hasToken;
  },

  dateFormat(date, format, localize = true) {
    if (!localize) {
      return dayjs.utc(date).tz(dayjs.tz.guess()).format(format);
    }
    return dayjs.utc(date).local().tz(dayjs.tz.guess()).format(format);
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

  // Converts string to titlecase i.e. 'hello' -> 'Hello'
  // @returns {String}
  toTitleCase(word) {
    let i, j, str, lowers, uppers;
    const _word = typeof word === "string" ? word : this;
    str = _word.replace(/([^\W_]+[^\s-]*) */g, (txt) => {
      return txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase();
    });

    // Certain minor words should be left lowercase unless
    // they are the first or last words in the string
    lowers = ["A", "An", "The", "And", "But", "Or", "For", "Nor", "As", "At",
      "By", "For", "From", "In", "Into", "Near", "Of", "On", "Onto", "To", "With"];
    for (i = 0, j = lowers.length; i < j; i++) {
      str = str.replace(new RegExp("\\s" + lowers[i] + "\\s", "g"), (txt) => {
        return txt.toLowerCase();
      });
    }

    // Certain words such as initialisms or acronyms should be left uppercase
    uppers = ["Id", "Tv"];
    for (i = 0, j = uppers.length; i < j; i++) {
      str = str.replace(new RegExp("\\b" + uppers[i] + "\\b", "g"), uppers[i].toUpperCase());
    }

    return str;
  },

  logoutUser(client) {
    const token = this.getToken();
    // TODO: for now we just remove the token,
    if (token) {
      if (client) { client.resetStore(); }
      window.localStorage.removeItem("token");
    }

    const sessionRoles = this.getSessionRoles();
    if (sessionRoles) {
      window.localStorage.removeItem("session_roles");
    }

    if (window.location.pathname !== "/secure-console") {
      window.location = "/secure-console";
    }
  },

  isEmailValid(email) {
    const newEmail = email.trim();
    const exp = /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
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
        currentPath = currentPath + "/" + part;
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
    if (bytes === 0) return "0 B";
    let i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
    if (i === 0) return bytes + " " + sizes[i];
    return (bytes / Math.pow(1024, i)).toFixed(1) + " " + sizes[i];
  },

  getDeployErrorTab(tabs) {
    if (trim(tabs["dryrunStderr"]) !== "") {
      return "dryrunStderr";
    } else if (trim(tabs["applyStderr"]) !== "") {
       return "applyStderr";
    } else {
      return Object.keys(tabs)[0];
    }
  },

  checkIsDateExpired(date) {
    const currentDate = dayjs();
    const diff = currentDate.diff(dayjs(date), "days");
    
    if(diff > 0) {
      return true;
    } else {
      return false;
    }
  },

  checkIsDeployedConfigLatest(app) {
    let latestSequence = app?.currentSequence;
    let deployedSequence = app?.downstreams[0]?.currentVersion?.parentSequence;
  
    if (deployedSequence === latestSequence) {
      return true;
    } else {
      return false;
    }
  }
};
