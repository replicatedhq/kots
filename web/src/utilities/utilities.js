import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";
import sortBy from "lodash/sortBy";
import jwt from "jsonwebtoken";
import cronstrue from "cronstrue";
import size from "lodash/size";
import { default as download } from "downloadjs";
dayjs.extend(utc);
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
    reader.readAsText(file);
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

/**
 * @param {String} - Returns the commit SHA of the current build
 */
export function getBuildVersion() {
  return window.env.SHIP_CLUSTER_BUILD_VERSION;
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
  case "0 0 1 * *":
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
      return "0 0 1 * *";
    default:
      return "0 0 * * MON";
  }
}

export function getReadableCronDescriptor(expression) {
  return cronstrue.toString(expression);
}

export function getServiceSite(provider) {
  const isGitlab = provider === "gitlab" || provider === "gitlab_enterprise";
  const isBitbucket = provider === "bitbucket" || provider === "bitbucket_server";
  const serviceSite = isGitlab ? "gitlab.com" : isBitbucket ? "bitbucket.org" : "github.com";
  return serviceSite;
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

export function getLineChanges(lineChangesArr) {
  let addedLines = 0;
  let removedLines = 0;
  lineChangesArr.forEach(lineChange => {
    const {
      originalStartLineNumber,
      originalEndLineNumber,
      modifiedStartLineNumber,
      modifiedEndLineNumber
    } = lineChange;

    if (originalEndLineNumber === originalStartLineNumber &&
      modifiedEndLineNumber === modifiedStartLineNumber &&
      originalEndLineNumber && modifiedEndLineNumber) {
      addedLines++;
      removedLines++;
    } else {
      if (modifiedEndLineNumber > modifiedStartLineNumber || originalEndLineNumber === 0) {
        addedLines += (modifiedEndLineNumber - modifiedStartLineNumber) + 1;
      }
      if (originalEndLineNumber > originalStartLineNumber || modifiedEndLineNumber === 0) {
        removedLines += (originalEndLineNumber - originalStartLineNumber) + 1;
      }
    }
  });
  return {
    addedLines: addedLines,
    removedLines: removedLines,
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

  getCookieValue(a) {
    var b = document.cookie.match("(^|[^;]+)\\s*" + a + "\\s*=\\s*([^;]+)");
    return b ? b.pop() : "";
  },

  isLoggedIn() {
    const hasToken = this.getToken();
    return !!hasToken;
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

    window.location = "/secure-console";
  },

  isEmailValid(email) {
    const newEmail = email.trim();
    const exp = /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    return exp.test(newEmail);
  },
};
