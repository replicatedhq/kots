/**
 * These functions are from React-Piwik
 * https://github.com/guillaumeparis2000/react-piwik/blob/master/src/React-Piwik.js
 */
class Tracker {
  constructor(options) {
    Tracker.push(["setSiteId", options.siteId]);
    Tracker.push(["setTrackerUrl", options.url]);
  }

  static push(args) {
    window._paq.push(args);
  }

  track = loc => {
    if (typeof window === "undefined") {
      return;
    }
    const currentPath = loc.path || (loc.pathname + loc.search).replace(/^\//, "");

    if (this.previousPath === currentPath) {
      return;
    }

    if (this.previousPath) {
      Tracker.push(["setReferrerUrl", `${window.location.origin}/${this.previousPath}`]);
    }
    Tracker.push(["setCustomUrl", `${window.location.origin}/${currentPath}`]);
    Tracker.push(["trackPageView"]);

    this.previousPath = currentPath;
  }

   connectToHistory(history) {
     const prevLoc = (typeof history.getCurrentLocation === "undefined") ? history.location : history.getCurrentLocation();
     this.previousPath = prevLoc.path || (prevLoc.pathname + prevLoc.search).replace(/^\//, "");

     history.listen(loc => {
       this.track(loc);
     });

     return history;
   }
}

export default function connectHistory(history) {
  if (!window._paq) {
    return history;
  }
  const tracker = new Tracker({
    siteId: 6,
    url: "https://data-2.replicated.com/js/"
  });
  return tracker.connectToHistory(history);
}
