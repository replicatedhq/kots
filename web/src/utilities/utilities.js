import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";
import pick from "lodash/pick";
import filter from "lodash/filter";
import sortBy from "lodash/sortBy";
import { default as download } from "downloadjs";
dayjs.extend(utc);
dayjs.extend(relativeTime);

/**
 *
 * @param {Object} - gitOpsRef - Object from GraphQL DB
 * @return {String} - "git" if a github deployment, otherwise "ship"
 */
export function getClusterType(gitOpsRef) {
  return gitOpsRef
    ? "git"
    : "ship";
}

/**
 * @param {Object} params - React Router History params object
 * @return {String} - slug of watch
 */
export function getCurrentWatch(params) {
  return params?.slug?.params?.slug;
}

/**
 * Takes a watched app object and returns its parsed metadata
 *
 * @param {String} watchMeta Metadata from watched app to parse
 * @return {Object}
 */
export function getWatchMetadata(watchMeta) {
  try {
    if (!watchMeta) return {};
    return JSON.parse(watchMeta);
  } catch (error) {
    console.error(error);
    return { error };
  }
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
 *
 * @param {String} -
 * @return {String} -
 */
export function getReadableLicenseType(type) {
  let readableType = "Development";
  if (type === "prod") {
    readableType = "Production"
  } else if (type === "trial") {
    readableType = "Trial"
  }
  return readableType;
}

/**
 * Retrieves the channel name the release is assigned to from app's stateJSON
 *
 * @param {String} watch The watched application to check
 * @return {String} name of the channel
 */
export function getAssignedReleaseChannel(stateJSON) {
  try {
    if (!stateJSON) return "";
    const state = JSON.parse(stateJSON);
    if (state?.v1?.upstreamContents?.appRelease) {
      return state.v1.upstreamContents.appRelease.channelName;
    }
    return "Unknown";
  } catch (error) {
    console.error(error);
    return "Unknown";
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

export function rootPath(path) {
  if (path[0] !== "/") {
    return path = "/"+path;
  } else {
    return path;
  }
}

export function getFileFormat(selectedFile) {
  if (selectedFile === "") {
    return "text";
  }
  if (selectedFile.includes(".json")) {
    return "json";
  } else if (selectedFile.includes(".human")) {
    return "yaml";
  }
  return "text";
}

/**
 * @param {Watch} watch - watch to determine type
 * @return {Boolean}
 */
export function isHelmChart(watch) {
  return Boolean(watch.helmName);
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
    } catch(e) {
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

  trimLeadingSlash(string) {
    if (string) {
      return `/${string.replace(/^\/+/g, "")}`;
    } else {
      return "";
    }
  },

  getNotificationType(item) {
    const keys = pick(item, ["email", "webhook", "pullRequest"]);
    const filteredKeys = filter(keys, (o) => o);
    if (filteredKeys[0] && filteredKeys[0].recipientAddress) {
      return "email";
    } else if (filteredKeys[0] && filteredKeys[0].uri) {
      return "webhook";
    } else {
      return "github";
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
    // this DOES NOT perform an actual logout of GitHub.
    if (token) {
      if (client) {client.resetStore();}
      window.localStorage.removeItem("token");
      window.location = "/login";
    } else {
      window.location = "/login";
    }
  },

  isEmailValid(email) {
    const newEmail = email.trim();
    const exp = /^(([^<>()[\]\\.,;:\s@"]+(\.[^<>()[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    return exp.test(newEmail);
  },

  async handleDownload(id) {
    const response = await fetch(`${window.env.SHIPDOWNLOAD_ENDPOINT}/${id}`, {
      headers: new Headers({
        "Authorization": Utilities.getToken(),
      }),
    })

    if (response.ok) {
      const blob = await response.blob();

      let contentType = response.headers.get("Content-Type");
      let filename = `rendered.${contentType}`;

      const contentDispositionHeader = response.headers.get("Content-Disposition");
      if (contentDispositionHeader) {
        ([, filename] = contentDispositionHeader.split("filename="));
      }

      download(blob, filename, response.headers.get("Content-Type"));
    }
  }
};
